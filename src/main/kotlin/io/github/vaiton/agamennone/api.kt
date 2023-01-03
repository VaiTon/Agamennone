package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.PartialFlag
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import io.ktor.server.application.*
import io.ktor.server.auth.*
import io.ktor.server.plugins.contentnegotiation.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.bson.conversions.Bson
import org.litote.kmongo.and
import org.litote.kmongo.eq
import org.litote.kmongo.gte
import org.litote.kmongo.lte
import java.time.LocalDateTime

fun Application.apiModule() {
    install(ContentNegotiation) {
        json()
    }
    configureAuth()
    configureRouting()
}

private fun Application.configureAuth() {
    val config = ConfigManager.config.value
    val serverPassword = config.serverApiPassword
        ?.takeUnless { it.isEmpty() }

    if (serverPassword == null) {
        log.debug("No server password set, disabling authentication")
        return
    }

    log.debug("Enabling authentication...")
    authentication {
        basic("api") {
            realm = "API"
            validate { credentials ->
                if (credentials.password == serverPassword) {
                    UserIdPrincipal("api")
                } else {
                    null
                }
            }
        }
    }
    log.debug("Authentication enabled.")
}

private fun Application.configureRouting() {
    routing {
        authenticate("api") {
            route("api") {
                prometheus()
                index()
                route("flags") {
                    getFlags()
                    postFlags()
                }
                getConfig()
            }
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

    val clientConfig = buildMap {
        put("flagRegex", config.flagRegex)
        teams?.let { put("teams", it) }
        flagInfo?.let { put("flagInfo", it) }
    }

    call.respond(clientConfig)
}

private fun Route.getFlags() = get {
    val conditions = mutableListOf<Bson>()
    call.request.queryParameters["from_cycle"]?.toIntOrNull()?.let {
        conditions += Flag::sentCycle gte it
    }
    call.request.queryParameters["to_cycle"]?.toIntOrNull()?.let {
        conditions += Flag::sentCycle lte it
    }
    call.request.queryParameters["from_time"]?.let {
        conditions += Flag::receivedTime gte LocalDateTime.parse(it)
    }
    call.request.queryParameters["to_time"]?.let {
        conditions += Flag::receivedTime lte LocalDateTime.parse(it)
    }
    call.request.queryParameters["team"]?.let {
        conditions += Flag::team eq it
    }
    call.request.queryParameters["status"]?.let {
        conditions += Flag::status eq FlagStatus.valueOf(it)
    }
    val limit = call.request.queryParameters["limit"]?.toIntOrNull()
    val filter = and(conditions)

    call.respond(FlagDatabase.getFlags(filter, limit))
}

private fun Route.postFlags() = post {
    val config = ConfigManager.config.value
    val flagRegex = config.flagRegex.toRegex()
    val receivedTime = LocalDateTime.now()

    val partialFlags = kotlin.runCatching {
        call.receive<List<PartialFlag>>()
    }.getOrElse {
        call.respond(HttpStatusCode.BadRequest, "Invalid request body")
        return@post
    }

    val flags = partialFlags
        .filter { it.isValid(flagRegex) }
        .map { Flag(it.flag, it.sploit, it.team, receivedTime, FlagStatus.QUEUED) }

    val insertedFlags = FlagDatabase.addFlags(flags)
    call.respond(HttpStatusCode.Created, insertedFlags)
}
