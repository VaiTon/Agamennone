package io.github.vaiton.agamennone.submit.protocols

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.submit.SubmissionProtocol
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import mu.KotlinLogging
import java.io.BufferedReader
import java.io.IOException
import java.io.OutputStreamWriter
import java.net.Socket

/**
 * [Protocol](https://ctf-gameserver.org/submission/)
 */
object EnoWars : SubmissionProtocol {
    private val log = KotlinLogging.logger {}
    private val WHITESPACE = Regex("[\\s\\t]")
    private val NEWLINE = Char(0x0A)

    override suspend fun submitFlags(
        flags: List<String>,
        config: Config,
    ): List<SubmissionProtocol.SubmissionResult> = withContext(Dispatchers.IO) {
        // The client connects to the server on a TCP port specified by
        // the respective CTF.
        log.debug {
            buildString {
                append("Connecting to ${config.submissionHost}:${config.submissionPort}")
                append(" with ${flags.size} flags")
                if (config.submissionPath != null) {
                    append(" and path ${config.submissionPath}")
                }
            }
        }
        // create the socket and auto-close it after use
        Socket(config.submissionHost, config.submissionPort).use { socket ->

            val writer = socket.getOutputStream().writer()
            val reader = socket.getInputStream().bufferedReader()

            // > The server MAY send a welcome banner, consisting of anything except
            // > two subsequent newlines.
            // > The server MUST indicate that the welcome sequence has finished by sending
            // > two subsequent newlines (\n\n).
            if (config.submissionSendsWelcomeBanner) {
                log.debug { "Reading welcome message.." }
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
                log.debug { welcomeMessage }
                log.debug { "Welcome message read." }
            }

            // Send flags and read responses
            return@withContext flags.map {
                sendFlag(writer, reader, it)
            }
        }
    }

    private fun sendFlag(
        writer: OutputStreamWriter,
        reader: BufferedReader,
        flag: String,
    ): SubmissionProtocol.SubmissionResult {
        // To submit a flag, the client MUST send the flag followed by a single newline.
        writer.write(flag + "\n")
        writer.flush()
        log.debug { "Sent flag '${flag}'" }

        // The server's response MUST consist of:
        //
        // - A repetition of the submitted flag
        // - Whitespace
        // - One of the response codes defined below
        // - Optionally: Whitespace, followed by a custom message consisting of any characters except newlines
        // - Newline

        val line = reader.readLine() ?: throw IOException("Server closed connection")
        log.debug { "Received response '$line'" }

        val parts = line.split(WHITESPACE)
        val responseCode = parts[1].trim()
        val status = when (responseCode) {
            "OK" -> FlagStatus.ACCEPTED
            "DUP", "OWN", "OLD" -> FlagStatus.SKIPPED
            "INV", "ERR" -> FlagStatus.REJECTED
            else -> throw IllegalStateException("Unknown response code: $responseCode")
        }

        val message = parts.getOrNull(2)?.trim()

        return SubmissionProtocol.SubmissionResult(flag, status, "$responseCode $message")
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
