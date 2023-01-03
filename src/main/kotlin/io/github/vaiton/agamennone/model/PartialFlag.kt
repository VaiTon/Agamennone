package io.github.vaiton.agamennone.model

import kotlinx.serialization.Serializable

@Serializable
data class PartialFlag(
    val flag: String,
    val sploit: String,
    val team: String
) {
    fun isValid(regex: Regex) = flag.isNotBlank() && flag.matches(regex)
}

