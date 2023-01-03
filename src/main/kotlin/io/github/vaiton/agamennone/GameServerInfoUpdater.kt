package io.github.vaiton.agamennone

import com.jayway.jsonpath.JsonPath
import io.github.vaiton.agamennone.model.Team
import io.ktor.client.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import mu.KotlinLogging
import kotlin.time.Duration.Companion.seconds


object GameServerInfoUpdater {
    private val _teams = MutableStateFlow<List<Team>?>(null)
    val teams = _teams.asStateFlow()

    private val _flagInfo = MutableStateFlow<Any?>(null)
    val flagInfo = _flagInfo.asStateFlow()

    private val log = KotlinLogging.logger {}
    private val client = HttpClient {
        install(ContentNegotiation) { json() }
    }

    suspend fun startUpdaters() {
        withContext(Dispatchers.Default) {
            launch { updateTeams() }
            launch { updateFlagInfo() }
        }
    }

    /**
     * WARNING: This coroutine runs forever.
     */
    private suspend fun updateFlagInfo() {
        while (true) {
            val config = ConfigManager.config.value
            val flagInfoUrl = config.flagInfoUrl
            if (flagInfoUrl == null) {
                log.trace { "No flag info URL provided. Not providing flag info." }
                return
            }
            val flagInfoQuery = checkNotNull(config.flagInfoQuery) { "flagInfoQuery is not set in config" }
            val flagInfoPeriod =
                checkNotNull(config.flagInfoRefreshPeriod) { "flagInfoRefreshPeriod is not set in config" }

            val response = client.get(flagInfoUrl).bodyAsText()

            val result: Any = JsonPath.parse(response).read(flagInfoQuery)
            _flagInfo.value = result

            delay(flagInfoPeriod.seconds)
        }
    }

    /**
     * WARNING: This coroutine runs forever.
     */
    private suspend fun updateTeams() {
        while (true) {
            // Update config
            val config = ConfigManager.config.value
            // Get team query
            val teamsInfoUrl = config.teamsInfoUrl
            if (teamsInfoUrl == null) {
                log.trace { "No teams provided. Not providing team info." }
                return
            }
            val teamsInfoQuery = checkNotNull(config.teamsInfoQuery) { "teamsInfoUrl is not set in config" }
            val teamsInfoPeriod =
                checkNotNull(config.teamsInfoRefreshPeriod) { "teamsInfoRefreshPeriod is not set in config" }

            val response = client.get(teamsInfoUrl).bodyAsText()

            val result: List<Team>? = JsonPath.parse(response).read(teamsInfoQuery)
            _teams.value = result

            delay(teamsInfoPeriod.seconds)
        }
    }
}