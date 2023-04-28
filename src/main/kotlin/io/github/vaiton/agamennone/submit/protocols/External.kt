package io.github.vaiton.agamennone.submit.protocols

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.submit.SubmissionProtocol
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromStream
import kotlin.io.path.Path
import kotlin.io.path.absolutePathString
import kotlin.io.path.exists
import kotlin.io.path.isExecutable

object External : SubmissionProtocol {

    @OptIn(ExperimentalSerializationApi::class)
    override suspend fun submitFlags(
        flags: List<String>,
        config: Config,
    ): List<SubmissionProtocol.SubmissionResult> = withContext(Dispatchers.IO) {

        // check if path is provided and executable
        val pathConfig = config.submissionExePath
            ?: error("No submitter path provided in config.")

        val submitterPath = Path(pathConfig)
            .takeIf { it.exists() && it.isExecutable() }
            ?: error("Submitter path '${pathConfig}' does not exist or is not executable.")


        // start process
        val process = ProcessBuilder(submitterPath.absolutePathString()).start()

        // write flags to stdin
        process.outputStream.bufferedWriter().use { out ->
            flags.forEach { flag ->
                out.write(flag)
                out.newLine()
            }
        }

        // wait for the process to finish
        val statusCode = process.waitFor()
        check(statusCode == 0) {
            "Submitter exited with status code $statusCode"
        }


        // read flags from stdout
        Json.decodeFromStream(process.inputStream)
    }
}