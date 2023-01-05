package io.github.vaiton.agamennone.model

import kotlinx.serialization.Serializable

@Serializable
data class Team(
    val name: String,
    val ip: String,
)