package io.github.vaiton.agamennone.api

import io.github.vaiton.agamennone.ConfigManager
import io.github.vaiton.agamennone.GameServerInfoUpdater
import io.github.vaiton.agamennone.compatibility.DestructiveFarm
import io.github.vaiton.agamennone.prometheus
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import io.ktor.server.application.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.server.plugins.cors.routing.*
import io.ktor.server.response.*
import io.ktor.server.routing.*

internal fun Application.apiModule() {
    install(ContentNegotiation) {
        json()
    }
    install(CORS) {
        allowMethod(HttpMethod.Options)
        allowMethod(HttpMethod.Put)
        allowMethod(HttpMethod.Delete)
        allowMethod(HttpMethod.Patch)
        allowHeader(HttpHeaders.Authorization)
        anyHost()
    }
    configureRouting()
}

private fun Application.configureRouting() {
    routing {
        index()
        prometheus()
        route("api") {
            getConfig()
            flagRoutes()
        }
    }
}

private fun Route.index() {
    get("/") {
        call.respond(HttpStatusCode.OK, "Agamennone API")
    }
}

private fun Route.getConfig() = get("config") {
    val config = ConfigManager.config.value
    val teams = GameServerInfoUpdater.teams.value
    val flagInfo = GameServerInfoUpdater.flagInfo.value

    if ("new" in call.request.queryParameters) {
        TODO("New Client-Server Protocol not implemented yet")
    } else {
        // Compatibility with old clients
        val response = DestructiveFarm.clientConfig(config, teams, flagInfo)
        call.respond(response)
    }
}

