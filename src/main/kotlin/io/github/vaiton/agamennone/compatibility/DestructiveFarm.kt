package io.github.vaiton.agamennone.compatibility

import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.model.Team
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement

object DestructiveFarm {

    fun getDestructiveFarmClientRequest(
        config: Config,
        teams: List<Team>,
        flagInfo: JsonElement?,
    ): JsonObject {
        val flagFormatJson = Json.encodeToJsonElement(config.flagRegex)
        val flagLifetimeJson = Json.encodeToJsonElement(config.flagLifetime)
        val submitPeriodJson = Json.encodeToJsonElement(config.submissionPeriod)

        val teamsMap: Map<String, String> = teams.associate { it.name to it.ip }
        val teamsJson = Json.encodeToJsonElement(teamsMap)

        val flagInfoJson = Json.encodeToJsonElement(flagInfo)

        return JsonObject(
            buildMap {
                put("FLAG_FORMAT", flagFormatJson)
                put("SUBMIT_PERIOD", submitPeriodJson)
                put("FLAG_LIFETIME", flagLifetimeJson)
                put("TEAMS", teamsJson)
                put("ATTACK_INFO", flagInfoJson)
            }
        )
    }
}