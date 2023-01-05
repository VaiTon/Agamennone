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
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.decodeFromStream
import mu.KotlinLogging
import kotlin.io.path.Path
import kotlin.io.path.exists
import kotlin.io.path.inputStream
import kotlin.reflect.KProperty1
import kotlin.time.Duration.Companion.seconds


object GameServerInfoUpdater {
    private val _teams = MutableStateFlow(getInitialTeams())

    @OptIn(ExperimentalSerializationApi::class)
    private fun getInitialTeams(): List<Team> {
        val teamsFile = Path("teams.json").takeIf { it.exists() } ?: error("No teams.json file found")
        val teamsMap = teamsFile.inputStream().use { stream ->
            Json.decodeFromStream<Map<String, String>>(stream)
        }
        return teamsMap.map { (name, ip) -> Team(name, ip) }
    }

    val teams = _teams.asStateFlow()

    private val _flagInfo = MutableStateFlow<JsonElement?>(null)
    val flagInfo = _flagInfo.asStateFlow()

    private val log = KotlinLogging.logger {}
    private val client = HttpClient {
        install(ContentNegotiation) { json() }
    }

    suspend fun startUpdaters() {
        withContext(Dispatchers.Default) {
            launch { updateTeamsInfo() }
            launch { updateFlagInfo() }
        }
    }

    /**
     * WARNING: This coroutine runs forever.
     */
    private suspend fun updateFlagInfo() {
        _flagInfo.updateGenericInfo(
            urlProperty = Config::flagInfoUrl,
            queryProperty = Config::flagInfoQuery,
            periodProperty = Config::flagInfoRefreshPeriod,
        )
    }

    /**
     * WARNING: This coroutine runs forever.
     */
    private suspend fun updateTeamsInfo() {
        _teams.updateGenericInfo(
            urlProperty = Config::teamsInfoUrl,
            queryProperty = Config::teamsInfoQuery,
            periodProperty = Config::teamsInfoRefreshPeriod
        )
    }

    private suspend fun <T> MutableStateFlow<T>.updateGenericInfo(
        urlProperty: KProperty1<Config, String?>,
        queryProperty: KProperty1<Config, String>,
        periodProperty: KProperty1<Config, Int>,
    ) {
        while (true) {
            val config = ConfigManager.config.value
            val url = urlProperty.get(config)
            if (url == null) {
                log.debug { "${urlProperty.name} not provided. Not providing info." }
                return
            }
            val query =
                checkNotNull(queryProperty.get(config)) { "${queryProperty.name} is not set in config" }
            val updatePeriod =
                checkNotNull(periodProperty.get(config)) { "${periodProperty.name} is not set in config" }

            val response = client.get(url).bodyAsText()

            val result: T? = JsonPath.parse(response).read<T>(query) // FIXME: This will never work...
            if (result == null) {
                log.warn { "Could not parse response from $url" }
            } else {
                log.debug { "Updating ${this@updateGenericInfo}" }
                this.value = result
            }

            delay(updatePeriod.seconds)
        }
    }
}