package com.remotelauncher.ui.apps

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.remotelauncher.data.AppsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import com.remotelauncher.net.LaunchResponse
import com.remotelauncher.net.PairResponse
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.ServerStatus
import com.remotelauncher.ui.theme.RemoteLauncherTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

private class FakeApi(
    private val appsResult: ApiResult<List<AppInfo>>,
) : RemoteLauncherApi {
    override suspend fun status(): ApiResult<ServerStatus> =
        ApiResult.NetworkError(RuntimeException("not used"))
    override suspend fun apps(token: String): ApiResult<List<AppInfo>> = appsResult
    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> =
        ApiResult.Success(PairResponse("tok"))
    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))
}

private class FakeTokenStore : TokenStore {
    private val map = mutableMapOf("https://example:8443" to "token")
    override fun getToken(serverUrl: String): String? = map[serverUrl]
    override fun setToken(serverUrl: String, token: String) { map[serverUrl] = token }
    override fun clearToken(serverUrl: String) { map.remove(serverUrl) }
}

@RunWith(AndroidJUnit4::class)
class AppsScreenTest {

    @get:Rule
    val composeTestRule = createComposeRule()

    private fun setContent(appsResult: ApiResult<List<AppInfo>>) {
        composeTestRule.setContent {
            val repo = AppsRepository(
                apiFactory = { FakeApi(appsResult) },
                tokenStore = FakeTokenStore(),
                serverUrl = "https://example:8443",
            )
            val vm = AppsViewModel(repo)
            RemoteLauncherTheme {
                AppsScreen(
                    viewModel = vm,
                    onUnauthorized = {},
                    onDisconnect = {},
                )
            }
        }
    }

    @Test
    fun topBar_showsTitle_andActionIcons() {
        setContent(ApiResult.Success(emptyList()))
        composeTestRule.waitForIdle()
        composeTestRule.onNodeWithText("Приложения").assertIsDisplayed()
        composeTestRule.onNodeWithText("На сервере нет приложений").assertIsDisplayed()
    }

    @Test
    fun loadedState_showsGridItems() {
        val apps = listOf(
            AppInfo(id = "firefox", name = "Firefox"),
            AppInfo(id = "obsidian", name = "Obsidian"),
            AppInfo(id = "vlc", name = "VLC media player"),
        )
        setContent(ApiResult.Success(apps))
        composeTestRule.waitForIdle()
        composeTestRule.onNodeWithText("Firefox").assertIsDisplayed()
        composeTestRule.onNodeWithText("Obsidian").assertIsDisplayed()
        composeTestRule.onNodeWithText("VLC media player").assertIsDisplayed()
    }

    @Test
    fun errorState_showsRetryButton() {
        setContent(ApiResult.HttpError(500))
        composeTestRule.waitForIdle()
        composeTestRule.onNodeWithText("Повторить").assertIsDisplayed()
    }
}
