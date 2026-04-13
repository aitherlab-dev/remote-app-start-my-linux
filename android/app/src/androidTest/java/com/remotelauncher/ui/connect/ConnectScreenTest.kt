package com.remotelauncher.ui.connect

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import com.remotelauncher.net.LaunchResponse
import com.remotelauncher.net.PairResponse
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.ServerStatus
import com.remotelauncher.ui.theme.RemoteLauncherTheme
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File
import java.util.UUID

private class FakeApi : RemoteLauncherApi {
    override suspend fun status(): ApiResult<ServerStatus> = ApiResult.NetworkError(RuntimeException("fake"))
    override suspend fun apps(token: String): ApiResult<List<AppInfo>> = ApiResult.Success(emptyList())
    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> =
        ApiResult.Success(PairResponse("token"))
    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))
}

private class FakeTokenStore : TokenStore {
    private val tokens = mutableMapOf<String, String>()
    private val pins = mutableMapOf<String, String>()
    override fun getToken(serverUrl: String): String? = tokens[serverUrl]
    override fun setToken(serverUrl: String, token: String) { tokens[serverUrl] = token }
    override fun clearToken(serverUrl: String) { tokens.remove(serverUrl) }
    override fun getPin(serverUrl: String): String? = pins[serverUrl]
    override fun setPin(serverUrl: String, pinHex: String) { pins[serverUrl] = pinHex }
    override fun clearPin(serverUrl: String) { pins.remove(serverUrl) }
}

@RunWith(AndroidJUnit4::class)
class ConnectScreenTest {

    @get:Rule
    val composeTestRule = createComposeRule()

    private lateinit var repo: SettingsRepository

    @Before
    fun setUp() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val file = File(context.cacheDir, "test-${UUID.randomUUID()}.preferences_pb")
        val store = PreferenceDataStoreFactory.create { file }
        repo = SettingsRepository(store)
    }

    private fun setContent() {
        composeTestRule.setContent {
            val vm = ConnectViewModel(repo, { FakeApi() }, FakeTokenStore())
            RemoteLauncherTheme {
                ConnectScreen(viewModel = vm)
            }
        }
    }

    @Test
    fun connectScreen_showsInputFieldAndButton() {
        setContent()
        composeTestRule.onNodeWithText("Подключиться").assertIsDisplayed()
        composeTestRule.onNodeWithText("Адрес сервера").assertIsDisplayed()
    }

    @Test
    fun enteringText_buttonRemainsEnabled() {
        setContent()
        val button = composeTestRule.onNodeWithText("Подключиться")
        button.assertIsEnabled()
        composeTestRule.onNodeWithText("Адрес сервера").performTextInput("localhost")
        button.assertIsEnabled()
    }

    @Test
    fun emptyInput_showsError() {
        setContent()
        composeTestRule.onNodeWithText("Подключиться").performClick()
        composeTestRule.onNodeWithText("адрес пустой").assertIsDisplayed()
    }
}
