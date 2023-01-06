package io.github.vaiton.agamennone.model

import kotlinx.serialization.Serializable
import java.time.LocalDateTime

@Serializable
data class Flag(
    val flag: String,
    val sploit: String,
    val team: String,
    @Serializable(with = LocalDateTimeSerializer::class)
    val receivedTime: LocalDateTime,
    val status: FlagStatus,
    val checkSystemResponse: String? = null,
    val sentCycle: Int? = null,
) {
    fun isValid(regex: Regex) = flag.isNotBlank() && flag.matches(regex)
}

