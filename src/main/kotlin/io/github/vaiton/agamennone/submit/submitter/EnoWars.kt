package io.github.vaiton.agamennone.submit.submitter

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.submit.SubmissionProtocol
import io.github.vaiton.agamennone.submit.SubmissionProtocol.SubmissionResult
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOn
import mu.KotlinLogging
import java.io.BufferedReader
import java.io.IOException
import java.io.OutputStreamWriter
import java.net.Socket

/**
 * [Protocol](https://ctf-gameserver.org/submission/)
 */
class EnoWars : SubmissionProtocol {
    private val log = KotlinLogging.logger {}
    private val WHITESPACE = Regex("[\\s\\t]")
    private val NEWLINE = Char(0x0A)

    override suspend fun submitFlags(
        flags: List<String>,
        config: Config,
    ): Flow<SubmissionResult> = flow {
        val submissionPort = checkNotNull(config.submissionPort) {
            "No submission port specified in config file."
        }

        // The client connects to the server on a TCP port specified by
        // the respective CTF.
        log.debug {
            buildString {
                append("Connecting to ${config.submissionHost}:$submissionPort")
                append(" with ${flags.size} flags")
            }
        }
        // create the socket and auto-close it after use
        Socket(config.submissionHost, submissionPort).use { socket ->

            val writer = socket.getOutputStream().writer()
            val reader = socket.getInputStream().bufferedReader()

            // > The server MAY send a welcome banner, consisting of anything except
            // > two subsequent newlines.
            // > The server MUST indicate that the welcome sequence has finished by sending
            // > two subsequent newlines (\n\n).
            if (config.submissionSendsWelcomeBanner) {
                log.trace { "Reading welcome message.." }
                val welcomeMessage = kotlin.runCatching {
                    readWelcomeBanner(reader)
                }.getOrElse { e ->
                    throw IOException(
                        "Failed to read welcome message. " +
                                "If this server does not send a welcome message please disable it " +
                                "in the config file.",
                        e
                    )

                }
                log.trace { welcomeMessage }
                log.trace { "Welcome message read." }
            }

            // Send flags and read responses
            for (flag in flags) {
                emit(sendFlag(writer, reader, flag))
            }
        }
    }.flowOn(Dispatchers.IO)

    private fun sendFlag(
        writer: OutputStreamWriter,
        reader: BufferedReader,
        flag: String,
    ): SubmissionResult {
        // To submit a flag, the client MUST send the flag followed by a single newline.
        writer.write(flag + "\n")
        writer.flush()
        log.trace { "Sent flag '${flag}'" }

        // The server's response MUST consist of:
        //
        // - A repetition of the submitted flag
        // - Whitespace
        // - One of the response codes defined below
        // - Optionally: Whitespace, followed by a custom message consisting of any characters except newlines
        // - Newline

        val line = reader.readLine() ?: throw IOException("Server closed connection")
        log.trace { "Received response '$line'" }

        val parts = line.split(WHITESPACE)
        val responseCode = parts[1].trim()
        val status = when (responseCode) {
            "OK" -> FlagStatus.ACCEPTED
            "DUP", "OWN", "OLD" -> FlagStatus.SKIPPED
            "INV", "ERR" -> FlagStatus.REJECTED
            else -> throw IllegalStateException("Unknown response code: $responseCode")
        }

        val message = parts.getOrNull(2)?.trim()

        return SubmissionResult(flag, status, "$responseCode $message")
    }

    private fun readWelcomeBanner(reader: BufferedReader): String {
        return buildString {
            // read until two newlines
            var newlines = 0
            while (newlines < 2) {
                val intChar = reader.read()

                // If a general error with the connection or its configuration
                // renders the server inoperable, it MAY send an arbitrary error
                // message and close the connection before sending the welcome sequence.
                // The error message MUST NOT contain two subsequent newlines.
                if (intChar == -1) {
                    throw IOException("Unexpected EOF. Got '$this'")
                }
                val char = intChar.toChar()
                if (char == NEWLINE) {
                    newlines++
                } else {
                    newlines = 0
                    append(char)
                }
            }
        }
    }
}
