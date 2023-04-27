package io.github.vaiton.agamennone.api

import io.github.vaiton.agamennone.compatibility.DestructiveFarm
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import kotlinx.serialization.Serializable
import org.jetbrains.exposed.sql.Op
import org.jetbrains.exposed.sql.and
import org.jetbrains.exposed.sql.transactions.experimental.newSuspendedTransaction
import java.time.LocalDateTime

internal fun Route.flagRoutes() {
    route("flags") {
        getFlags()
        postFlags()
    }
}

@Serializable
data class FlagsResponse(
    val flag: String,
    val sploit: String,
    val team: String,
    val receivedTime: String,
    val status: FlagStatus,
    val checkSystemResponse: String?,
    val sentCycle: Int?,
) {
    constructor(flag: Flag) : this(
        flag.flag,
        flag.sploit,
        flag.team,
        flag.receivedTime.toString(),
        flag.status,
        flag.checkSystemResponse,
        flag.sentCycle,
    )
}

private fun Route.getFlags() = get {
    var filter: Op<Boolean> = Op.TRUE
    call.request.queryParameters["from_cycle"]?.toIntOrNull()?.let {
        filter = filter.and { Flags.sentCycle greaterEq it }
    }
    call.request.queryParameters["to_cycle"]?.toIntOrNull()?.let {
        filter = filter.and { Flags.sentCycle lessEq it }
    }
    call.request.queryParameters["from_time"]?.let {
        filter = filter.and { Flags.receivedTime greaterEq LocalDateTime.parse(it) }
    }
    call.request.queryParameters["to_time"]?.let {
        filter = filter.and { Flags.receivedTime lessEq LocalDateTime.parse(it) }
    }
    call.request.queryParameters["team"]?.let {
        filter = filter.and { Flags.team eq it }
    }
    call.request.queryParameters["status"]?.let {
        filter = filter.and { Flags.status eq FlagStatus.valueOf(it) }
    }
    val limit = call.request.queryParameters["limit"]?.toIntOrNull()


    val flags = newSuspendedTransaction {
        Flag.find(filter)
            .apply { if (limit != null) limit(limit) }
            .map(::FlagsResponse)
    }
    call.respond(flags)
}

private fun Route.postFlags() = post {
    DestructiveFarm.clientFlags(this)
    call.respond(HttpStatusCode.Created)
}

