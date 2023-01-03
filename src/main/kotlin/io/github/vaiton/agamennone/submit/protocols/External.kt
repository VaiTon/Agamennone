package io.github.vaiton.agamennone.submit.protocols

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.submit.SubmissionProtocol
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromStream
import kotlinx.serialization.json.encodeToStream
import mu.KotlinLogging
import kotlin.io.path.Path
import kotlin.io.path.absolutePathString
import kotlin.io.path.exists
import kotlin.io.path.isExecutable

object External : SubmissionProtocol {
    private val log = KotlinLogging.logger {}

    @OptIn(ExperimentalSerializationApi::class)
    override suspend fun submitFlags(
        flags: List<Flag>,
        config: Config,
    ): List<Flag> = withContext(Dispatchers.IO) {
        val pathConfig = config.submissionExePath ?: error("No submitter path provided in config.")
        val submitterPath = Path(pathConfig)

        if (!submitterPath.exists() || !submitterPath.isExecutable()) {
            log.error { "Submitter path '${submitterPath.absolutePathString()}' does not exist or is not executable." }
            return@withContext emptyList()
        }

        val process = ProcessBuilder(submitterPath.absolutePathString()).start()

        // write flags to stdin
        process.outputStream.use { Json.encodeToStream(flags, it) }

        // wait for the process to finish
        val statusCode = process.waitFor()

        if (statusCode == 0) {
            // read flags from stdout
            Json.decodeFromStream(process.inputStream)
        } else {
            log.error { "Submitter exited with status code $statusCode" }
            log.error { process.errorStream.reader().use { it.readText() } }
            emptyList()
        }
    }
}