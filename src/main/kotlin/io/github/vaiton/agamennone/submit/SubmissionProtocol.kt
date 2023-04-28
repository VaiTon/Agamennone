package io.github.vaiton.agamennone.submit

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.submit.submitter.CyberChallengeIT
import io.github.vaiton.agamennone.submit.submitter.EnoWars
import io.github.vaiton.agamennone.submit.submitter.External
import kotlinx.coroutines.flow.Flow
import kotlinx.serialization.Serializable
import kotlin.reflect.full.createInstance

interface SubmissionProtocol {

    @Serializable
    data class SubmissionResult(
        val flag: String,
        val status: FlagStatus,
        val checkSystemResponse: String,
    )

    /**
     * @return The submitted flags with updated status.
     */
    suspend fun submitFlags(flags: List<String>, config: Config): Flow<SubmissionResult>

    companion object {
        private val PROTOCOLS_MAP = mapOf(
            "ENOWARS" to EnoWars::class,
            "EXTERNAL" to External::class,
            "CYBERCHALLENGEIT" to CyberChallengeIT::class,
        )

        fun createProtocol(protocol: String): SubmissionProtocol =
            PROTOCOLS_MAP[protocol]?.createInstance() ?: error("Unknown protocol '$protocol'")
    }
}

