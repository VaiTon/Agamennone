package io.github.vaiton.agamennone.compatibility

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.ConfigManager
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Team
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.util.pipeline.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.*
import org.jetbrains.annotations.Contract
import java.time.LocalDateTime

internal object DestructiveFarm {
    @Contract(pure = true)
    fun clientConfig(
        config: Config,
        teams: List<Team>,
        flagInfo: JsonElement?,
    ): JsonObject = buildJsonObject {
        put("FLAG_FORMAT", config.flagRegex)
        put("SUBMIT_PERIOD", config.submissionPeriod)
        put("FLAG_LIFETIME", config.flagLifetime)
        putJsonObject("TEAMS") {
            teams.forEach { team ->
                put(team.name, team.ip)
            }
        }
        flagInfo?.let { put("ATTACK_INFO", it) }
    }
    @Serializable
    private data class PartialFlag(
        val flag: String,
        val sploit: String,
        val team: String
    )
    suspend fun clientFlags(context: PipelineContext<Unit, ApplicationCall>): List<Flag>? {
        val receivedTime = LocalDateTime.now()

        val config = ConfigManager.config.value
        val flagRegex = config.flagRegex.toRegex()

        val partialFlags = kotlin.runCatching {
            context.call.receive<List<PartialFlag>>()
        }.getOrElse {
            context.call.respond(HttpStatusCode.BadRequest, "Invalid request body")
            return null
        }

        return partialFlags
            .map { Flag(it.flag, it.sploit, it.team, receivedTime, FlagStatus.QUEUED) }
            .filter { it.isValid(flagRegex) }
    }
}