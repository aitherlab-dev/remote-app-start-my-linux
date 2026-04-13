package com.remotelauncher.net

import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

// Release-variant HttpClient factory. Uses the platform default TrustManager
// (which rejects self-signed certs) — release APKs have no dev-trust
// shortcut. Proper SPKI pinning lands in A5.1; until then release builds
// cannot talk to a self-signed local dev server.
fun createAppHttpClient(): HttpClient {
    return HttpClient(CIO) {
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true })
        }
        install(HttpTimeout) {
            requestTimeoutMillis = 10_000
        }
    }
}
