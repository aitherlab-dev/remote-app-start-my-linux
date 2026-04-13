package com.remotelauncher.ui.pairing

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
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

private class FakeApi : RemoteLauncherApi {
    override suspend fun status(): ApiResult<ServerStatus> =
        ApiResult.NetworkError(RuntimeException("not used"))
    override suspend fun apps(token: String): ApiResult<List<AppInfo>> =
        ApiResult.Success(emptyList())
    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> =
        ApiResult.Success(PairResponse("token"))
    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))
}

private class FakeTokenStore : TokenStore {
    private val map = mutableMapOf<String, String>()
    override fun getToken(serverUrl: String): String? = map[serverUrl]
    override fun setToken(serverUrl: String, token: String) { map[serverUrl] = token }
    override fun clearToken(serverUrl: String) { map.remove(serverUrl) }
}

@RunWith(AndroidJUnit4::class)
class PairingScreenTest {

    @get:Rule
    val composeTestRule = createComposeRule()

    private fun setContent() {
        composeTestRule.setContent {
            val vm = PairingViewModel(
                serverUrl = "https://example:8443",
                apiFactory = { FakeApi() },
                tokenStore = FakeTokenStore(),
                deviceLabel = "test-device",
            )
            RemoteLauncherTheme {
                PairingScreen(viewModel = vm, onPaired = {})
            }
        }
    }

    @Test
    fun pairingScreen_showsTitle_hint_pinField_andButton() {
        setContent()
        composeTestRule.onNodeWithText("Введите PIN с сервера").assertIsDisplayed()
        composeTestRule.onNodeWithText("PIN показан в терминале, где запущен RemoteLauncher")
            .assertIsDisplayed()
        composeTestRule.onNodeWithText("PIN").assertIsDisplayed()
        composeTestRule.onNodeWithText("Подтвердить").assertIsDisplayed()
    }

    @Test
    fun buttonDisabled_untilSixDigitsEntered() {
        setContent()
        val submit = composeTestRule.onNodeWithText("Подтвердить")
        submit.assertIsNotEnabled()
        composeTestRule.onNodeWithText("PIN").performTextInput("12345")
        submit.assertIsNotEnabled()
    }

    @Test
    fun buttonEnabled_whenSixDigitsEntered() {
        setContent()
        composeTestRule.onNodeWithText("PIN").performTextInput("123456")
        composeTestRule.onNodeWithText("Подтвердить").assertIsEnabled()
    }
}
