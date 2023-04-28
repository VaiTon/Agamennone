package io.github.vaiton.agamennone.submit.submitter

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.submit.SubmissionProtocol
import io.github.vaiton.agamennone.submit.SubmissionProtocol.SubmissionResult
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.request.*
import io.ktor.http.*
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

class CyberChallengeIT : SubmissionProtocol {
    private val client = HttpClient()

    override suspend fun submitFlags(flags: List<String>, config: Config): Flow<SubmissionResult> = flow {
        val token = checkNotNull(config.submissionToken) {
            "Submission Token cannot be null. Check your config.json"
        }
        val url = Url(checkNotNull(config.submissionUrl) {
            "Submission URL cannot be null. Check your config.json"
        })

        // send the http request
        val body = Json.encodeToString(flags)
        val responses: List<GameServerResponse> = client.put(url) {
            header("X-Team-Token", token)
            setBody(body)
        }.body()


        // map responses to SubmissionResults
        responses.forEach { response ->

            val msg = response.msg
            val msgTrimmed = msg.removePrefix("[${response.flag}] ")

            val status = RESPONSES_MAP.entries
                .find { entry -> entry.value.any { msgTrimmed.contains(it) } }
                ?.key
                ?: throw IllegalStateException("Cannot map message to a FlagStatus: '${msgTrimmed}'")

            emit(SubmissionResult(flag = response.flag, status = status, message = msgTrimmed))
        }
    }

    companion object {
        val RESPONSES_MAP = mapOf(
            FlagStatus.QUEUED to listOf(
                "timeout",
                "game not started",
                "try again later",
                "game over",
                "is not up"
            ),
            FlagStatus.ACCEPTED to listOf(
                "accepted",
                "congrat"
            ),
            FlagStatus.REJECTED to listOf(
                "bad",
                "wrong",
                "expired",
                "unknown",
                "your own",
                "too old",
                "not in database",
                "already submitted",
                "invalid flag",
                "denied",
                "no such flag"
            ),
        )

        data class GameServerResponse(
            val msg: String,
            val flag: String,
            val status: Boolean,
        )
    }

}