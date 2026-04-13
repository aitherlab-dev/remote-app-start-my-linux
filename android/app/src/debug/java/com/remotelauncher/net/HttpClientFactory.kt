package com.remotelauncher.net

import android.util.Log
import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

// TODO(A5.1): DELETE THIS FILE, replace with a pinned HttpClient factory.
//
// Debug-variant factory that wires a permissive [DevTrustManager] into the
// Ktor CIO engine so the client can talk to our self-signed server during
// development of A3/A4. Logs a visible warning every time the client is
// created so any accidental leak into a shipping build is obvious in
// logcat. The release-variant of this file lives under src/release/java/
// and does NOT use DevTrustManager.
fun createAppHttpClient(): HttpClient {
    Log.w(
        "RemoteLauncher",
        "Using DEV HttpClient with permissive TLS — must be replaced in A5.1",
    )
    return HttpClient(CIO) {
        engine {
            https {
                trustManager = DevTrustManager()
            }
        }
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true })
        }
        install(HttpTimeout) {
            requestTimeoutMillis = 10_000
        }
    }
}
