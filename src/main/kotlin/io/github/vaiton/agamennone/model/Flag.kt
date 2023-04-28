package io.github.vaiton.agamennone.model

import org.jetbrains.exposed.dao.IntEntity
import org.jetbrains.exposed.dao.IntEntityClass
import org.jetbrains.exposed.dao.id.EntityID
import org.jetbrains.exposed.dao.id.IntIdTable
import org.jetbrains.exposed.sql.javatime.datetime

object Flags : IntIdTable() {
    val flag = varchar("flag", 255).uniqueIndex()
    val sploit = varchar("sploit", 255)
    val team = varchar("team", 255)
    val receivedTime = datetime("received_time")
    val status = enumeration("status", FlagStatus::class)
    val checkSystemResponse = varchar("check_system_response", 255).nullable()
    val sentCycle = integer("sent_cycle").nullable()
}


class Flag(id: EntityID<Int>) : IntEntity(id) {
    companion object : IntEntityClass<Flag>(Flags)

    var flag by Flags.flag
    var sploit by Flags.sploit
    var team by Flags.team
    var receivedTime by Flags.receivedTime
    var status by Flags.status
    var checkSystemResponse by Flags.checkSystemResponse
    var sentCycle by Flags.sentCycle
}

