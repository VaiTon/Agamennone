package io.github.vaiton.agamennone.submit

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.submit.protocols.EnoWars
import io.github.vaiton.agamennone.submit.protocols.External
import kotlinx.serialization.Serializable

interface SubmissionProtocol {

    @Serializable
    data class SubmissionResult(
        val flag: String,
        val status: FlagStatus,
        val message: String,
    )

    /**
     * @return The submitted flags with updated status.
     */
    suspend fun submitFlags(flags: List<String>, config: Config): List<SubmissionResult>

    companion object {
        private val PROTOCOLS_MAP = mapOf(
            "ENOWARS" to EnoWars,
            "EXTERNAL" to External,
        )

        fun getProtocol(protocol: String): SubmissionProtocol =
            PROTOCOLS_MAP[protocol] ?: error("Unknown protocol '$protocol'")
    }
}

