package io.github.vaiton.agamennone

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromStream
import mu.KotlinLogging
import java.nio.file.*
import java.nio.file.StandardWatchEventKinds.OVERFLOW
import kotlin.io.path.Path
import kotlin.io.path.inputStream
import kotlin.io.path.name
import kotlin.time.Duration.Companion.seconds

@Serializable
data class Config(
    val gameName: String,

    val flagLifetime: Int = 500,
    val flagRegex: String,

    val submissionProtocol: String,
    val submissionHost: String,
    val submissionPort: Int,
    val submissionSendsWelcomeBanner: Boolean = true,
    val submissionPeriod: Int = 30,
    val submissionFlagLimit: Int = 100,
    val submissionPath: String? = null,
    val submissionExePath: String? = null,

    val flagInfoUrl: String? = null,
    val flagInfoQuery: String? = null,
    val flagInfoRefreshPeriod: Int = 30,

    val teamsInfoUrl: String? = null,
    val teamsInfoQuery: String? = null,
    val teamsInfoRefreshPeriod: Int = 30,

    val serverHost: String,
    val serverPort: Int,
    val serverApiPassword: String? = null,
)

@OptIn(ExperimentalSerializationApi::class)
object ConfigManager {
    private val log = KotlinLogging.logger {}
    private val json = Json { ignoreUnknownKeys = true }
    private val watchService: WatchService = FileSystems.getDefault().newWatchService()
    private val pathToWatch = Path("")

    private lateinit var _config: MutableStateFlow<Config>
    val config by lazy { _config.asStateFlow() } // We need lazy because config isn't initialized in the constructor

    private const val CONFIG_FILE = "config.json"

    /**
     * Wait for the config file to be changed by registering a watch service and then update the config.
     *
     * **WARNING: This function never returns.**
     */
    suspend fun updateOnConfigUpdate(): Nothing {
        withContext(Dispatchers.IO) {
            val pollKey = pathToWatch.register(
                watchService,
                StandardWatchEventKinds.ENTRY_CREATE,
                StandardWatchEventKinds.ENTRY_MODIFY
            )

            while (true) {
                for (event in pollKey.pollEvents()) {
                    val kind = event.kind()
                    if (kind == OVERFLOW) {
                        continue
                    }

                    @Suppress("UNCHECKED_CAST")
                    val pathEvent = event as WatchEvent<Path>
                    val path = pathEvent.context()

                    val child = pathToWatch.resolve(path)
                    if (child.name == CONFIG_FILE) {
                        log.debug { "Config file was modified." }
                        reloadConfig()
                        log.info { "Config reloaded." }
                    }
                }
                delay(1.seconds)
            }
        }
    }


    fun reloadConfig() {
        val newConfig = Path(CONFIG_FILE)
            .inputStream()
            .use { json.decodeFromStream<Config>(it) }

        if (!this::_config.isInitialized) {
            _config = MutableStateFlow(newConfig)
        } else {
            _config.value = newConfig
        }
    }
}