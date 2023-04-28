package io.github.vaiton.agamennone.compatibility

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.ConfigManager
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import io.github.vaiton.agamennone.model.Team
import io.ktor.server.application.*
import io.ktor.server.plugins.*
import io.ktor.server.request.*
import io.ktor.util.pipeline.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.*
import mu.KotlinLogging
import org.jetbrains.annotations.Contract
import org.jetbrains.exposed.sql.batchInsert
import org.jetbrains.exposed.sql.transactions.experimental.newSuspendedTransaction
import java.time.LocalDateTime

internal object DestructiveFarm {
    private val logger = KotlinLogging.logger {}

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
        put("ATTACK_INFO", flagInfo ?: JsonObject(emptyMap()))
    }

    @Serializable
    private data class PartialFlag(
        val flag: String,
        val sploit: String,
        val team: String,
    ) {
        fun isValid(regex: Regex) = flag.isNotBlank() && flag.matches(regex)
    }

    suspend fun clientFlags(context: PipelineContext<Unit, ApplicationCall>) {
        val receivedTime = LocalDateTime.now()

        val config = ConfigManager.config.value
        val flagRegex = config.flagRegex.toRegex()

        val partialFlags = context.call.receive<List<PartialFlag>>()
            .filter { it.isValid(flagRegex) } // Only accept valid flags


        logger.info {
            val remoteHost = context.call.request.origin.remoteHost
            val size = partialFlags.size
            "Received $size flags from DestructiveFarm client at $remoteHost"
        }

        // Insert flags into database
        newSuspendedTransaction {
            Flags.batchInsert(partialFlags, ignore = true) {
                this[Flags.flag] = it.flag
                this[Flags.sploit] = it.sploit
                this[Flags.team] = it.team
                this[Flags.receivedTime] = receivedTime
                this[Flags.status] = FlagStatus.QUEUED
            }
        }
    }
}