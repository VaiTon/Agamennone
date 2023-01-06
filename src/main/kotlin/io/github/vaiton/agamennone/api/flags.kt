package io.github.vaiton.agamennone.api

import io.github.vaiton.agamennone.FlagDatabase
import io.github.vaiton.agamennone.compatibility.DestructiveFarm
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.bson.conversions.Bson
import org.litote.kmongo.and
import org.litote.kmongo.eq
import org.litote.kmongo.gte
import org.litote.kmongo.lte
import java.time.LocalDateTime

internal fun Route.flagRoutes() {
    route("flags") {
        getFlags()
        postFlags()
    }
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
    val flags = FlagDatabase.getFlags(filter, limit)

    call.respond(flags)
}

private fun Route.postFlags() = post {
    val flags = DestructiveFarm.clientFlags(this) ?: return@post

    val insertedFlags = FlagDatabase.addFlags(flags)
    call.respond(HttpStatusCode.Created, insertedFlags)
}

