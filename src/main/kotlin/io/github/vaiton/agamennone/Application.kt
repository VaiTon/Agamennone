package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.api.apiModule
import io.github.vaiton.agamennone.storage.FlagDatabase
import io.github.vaiton.agamennone.submit.Submitter
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

val scope = CoroutineScope(SupervisorJob())

suspend fun main() {
    FlagDatabase.init()
    ConfigManager.reloadConfig()

    scope.launch { Submitter.loop() }
    scope.launch { ConfigManager.updateOnConfigUpdate() }
    scope.launch { GameServerInfoUpdater.startUpdaters() }

    val config = ConfigManager.config.value

    embeddedServer(
        Netty,
        host = config.serverHost,
        port = config.serverPort,
        module = Application::apiModule
    ).start(wait = true)

    scope.cancel()
}

