@Suppress("DSL_SCOPE_VIOLATION") // See https://github.com/gradle/gradle/issues/22797

plugins {
    alias(libs.plugins.kotlin)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.ktor)
    application
}

group = "io.github.vaiton"
version = "0.0.1"

application {
    mainClass.set("io.github.vaiton.ApplicationKt")

    val isDevelopment: Boolean = project.ext.has("development")
    applicationDefaultJvmArgs = listOf("-Dio.ktor.development=$isDevelopment")
}

repositories {
    mavenCentral()
}

dependencies {
    // Bundles
    implementation(libs.bundles.ktor.server)
    implementation(libs.bundles.ktor.client)
    implementation(libs.bundles.logback)
    implementation(libs.bundles.prometheus)

    // Single libraries
    implementation(libs.ktor.serialization.kotlinx.json)
    implementation(libs.jsonpath)
    implementation(libs.kmongo.coroutine)
    implementation(libs.kotlin.logging.jvm)
    implementation("io.ktor:ktor-client-content-negotiation:2.2.1")
    implementation("io.ktor:ktor-serialization-jackson:2.2.1")

    // Test
    testImplementation(libs.bundles.test)
}

kotlin {
    jvmToolchain(17)
}

application {
    mainClass.set("io.github.vaiton.agamennone.ApplicationKt")
}

