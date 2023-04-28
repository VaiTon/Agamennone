package io.github.vaiton.agamennone.api

import io.github.vaiton.agamennone.ConfigManager
import io.github.vaiton.agamennone.FlagDatabase
import io.github.vaiton.agamennone.GameServerInfoUpdater
import io.github.vaiton.agamennone.compatibility.DestructiveFarm
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import io.github.vaiton.agamennone.prometheus
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import io.ktor.server.application.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.server.plugins.cors.routing.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import kotlinx.serialization.Serializable
import org.jetbrains.exposed.sql.SqlExpressionBuilder.eq
import org.jetbrains.exposed.sql.transactions.experimental.newSuspendedTransaction

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
            getStats()
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

@Serializable
data class StatsResponse(
    val flags: Int,
    val queuedFlags: Int,
    val acceptedFlags: Int,
    val rejectedFlags: Int,
    val skippedFlags: Int,
    val flagsSentLastCycle: Int? = null,
    val lastCycle: Int? = null,
)

private fun Route.getStats() = get("stats") {
    val stats = newSuspendedTransaction {
        val lastCycle = FlagDatabase.getMaxCycle()
        val flagsSentLastCycle = lastCycle?.let {
            Flag.count(Flags.sentCycle eq it).toInt()
        } ?: 0

        StatsResponse(
            flags = Flag.count().toInt(),
            queuedFlags = Flag.count(Flags.status eq FlagStatus.QUEUED).toInt(),
            acceptedFlags = Flag.count(Flags.status eq FlagStatus.ACCEPTED).toInt(),
            rejectedFlags = Flag.count(Flags.status eq FlagStatus.REJECTED).toInt(),
            skippedFlags = Flag.count(Flags.status eq FlagStatus.SKIPPED).toInt(),
            flagsSentLastCycle = flagsSentLastCycle,
            lastCycle = lastCycle
        )
    }

    call.respond(stats)
}

