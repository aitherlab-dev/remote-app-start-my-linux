package com.remotelauncher

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.settingsDataStore
import com.remotelauncher.net.KtorRemoteLauncherApi
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.ui.AppNavHost
import com.remotelauncher.ui.theme.RemoteLauncherTheme
import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

class MainActivity : ComponentActivity() {

    private val httpClient: HttpClient by lazy {
        HttpClient(CIO) {
            install(ContentNegotiation) {
                json(Json { ignoreUnknownKeys = true })
            }
        }
    }

    private val apiFactory: (String) -> RemoteLauncherApi = { baseUrl ->
        KtorRemoteLauncherApi(httpClient, baseUrl)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val repository = SettingsRepository(applicationContext.settingsDataStore)
        setContent {
            RemoteLauncherTheme {
                AppNavHost(
                    settingsRepository = repository,
                    apiFactory = apiFactory,
                )
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        httpClient.close()
    }
}
