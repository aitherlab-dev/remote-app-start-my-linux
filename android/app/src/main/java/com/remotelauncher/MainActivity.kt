package com.remotelauncher

import android.os.Build
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import com.remotelauncher.data.EncryptedTokenStore
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
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
        val tokenStore: TokenStore = try {
            EncryptedTokenStore(applicationContext)
        } catch (t: Throwable) {
            Log.e(TAG, "Failed to init EncryptedTokenStore, falling back to in-memory", t)
            InMemoryTokenStore()
        }
        val deviceLabel = "${Build.MANUFACTURER} ${Build.MODEL}"
        setContent {
            RemoteLauncherTheme {
                AppNavHost(
                    settingsRepository = repository,
                    apiFactory = apiFactory,
                    tokenStore = tokenStore,
                    deviceLabel = deviceLabel,
                )
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        httpClient.close()
    }

    private class InMemoryTokenStore : TokenStore {
        private val map = mutableMapOf<String, String>()
        override fun getToken(serverUrl: String): String? = map[serverUrl]
        override fun setToken(serverUrl: String, token: String) { map[serverUrl] = token }
        override fun clearToken(serverUrl: String) { map.remove(serverUrl) }
    }

    companion object {
        private const val TAG = "MainActivity"
    }
}
