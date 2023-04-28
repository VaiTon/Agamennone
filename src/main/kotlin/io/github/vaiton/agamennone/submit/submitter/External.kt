package io.github.vaiton.agamennone.submit.submitter

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.submit.SubmissionProtocol
import io.github.vaiton.agamennone.submit.SubmissionProtocol.SubmissionResult
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOn
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromStream
import kotlin.io.path.Path
import kotlin.io.path.absolutePathString
import kotlin.io.path.exists
import kotlin.io.path.isExecutable

class External : SubmissionProtocol {

    @OptIn(ExperimentalSerializationApi::class)
    override suspend fun submitFlags(
        flags: List<String>,
        config: Config,
    ): Flow<SubmissionResult> = flow {

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
        val results: List<SubmissionResult> = Json.decodeFromStream(process.inputStream)
        results.forEach { emit(it) }
    }.flowOn(Dispatchers.IO)
}