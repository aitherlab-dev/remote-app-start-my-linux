package com.remotelauncher

import android.os.Build
import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import coil3.ImageLoader
import coil3.SingletonImageLoader
import coil3.network.ktor3.KtorNetworkFetcherFactory
import coil3.request.CachePolicy
import coil3.request.crossfade
import coil3.svg.SvgDecoder
import com.remotelauncher.data.EncryptedTokenStore
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.data.settingsDataStore
import com.remotelauncher.net.KtorRemoteLauncherApi
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.createAppHttpClient
import com.remotelauncher.ui.AppNavHost
import com.remotelauncher.ui.theme.RemoteLauncherTheme
import io.ktor.client.HttpClient

class MainActivity : ComponentActivity() {

    private val httpClient: HttpClient by lazy { createAppHttpClient() }

    private val apiFactory: (String) -> RemoteLauncherApi = { baseUrl ->
        KtorRemoteLauncherApi(httpClient, baseUrl)
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        SingletonImageLoader.setSafe { ctx ->
            ImageLoader.Builder(ctx)
                .components {
                    add(KtorNetworkFetcherFactory(httpClient))
                    add(SvgDecoder.Factory())
                }
                .crossfade(true)
                .diskCachePolicy(CachePolicy.ENABLED)
                .memoryCachePolicy(CachePolicy.ENABLED)
                .build()
        }

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
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background,
                ) {
                    AppNavHost(
                        settingsRepository = repository,
                        apiFactory = apiFactory,
                        tokenStore = tokenStore,
                        deviceLabel = deviceLabel,
                    )
                }
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        httpClient.close()
    }

    private class InMemoryTokenStore : TokenStore {
        private val tokens = mutableMapOf<String, String>()
        private val pins = mutableMapOf<String, String>()
        override fun getToken(serverUrl: String): String? = tokens[serverUrl]
        override fun setToken(serverUrl: String, token: String) { tokens[serverUrl] = token }
        override fun clearToken(serverUrl: String) { tokens.remove(serverUrl) }
        override fun getPin(serverUrl: String): String? = pins[serverUrl]
        override fun setPin(serverUrl: String, pinHex: String) { pins[serverUrl] = pinHex.uppercase() }
        override fun clearPin(serverUrl: String) { pins.remove(serverUrl) }
    }

    companion object {
        private const val TAG = "MainActivity"
    }
}
