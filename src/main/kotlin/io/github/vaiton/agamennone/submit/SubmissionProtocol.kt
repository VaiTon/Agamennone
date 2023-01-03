package io.github.vaiton.agamennone.submit

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.submit.protocols.EnoWars
import io.github.vaiton.agamennone.submit.protocols.External

interface SubmissionProtocol {

    /**
     * @return The submitted flags with updated status.
     */
    suspend fun submitFlags(flags: List<Flag>, config: Config): List<Flag>

    companion object {
        private val PROTOCOLS_MAP = mapOf(
            "ENOWARS" to EnoWars,
            "EXTERNAL" to External,
        )

        fun getProtocol(protocol: String): SubmissionProtocol =
            PROTOCOLS_MAP[protocol] ?: error("Unknown protocol '$protocol'")
    }
}
