package com.remotelauncher.ui.apps

import com.remotelauncher.data.AppsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import com.remotelauncher.net.LaunchResponse
import com.remotelauncher.net.PairResponse
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.ServerStatus
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

private class FakeApi(
    var appsResult: ApiResult<List<AppInfo>> = ApiResult.Success(emptyList()),
) : RemoteLauncherApi {
    var appsCalls: Int = 0
        private set
    var lastToken: String? = null
        private set

    override suspend fun status(): ApiResult<ServerStatus> =
        ApiResult.NetworkError(RuntimeException("not used"))

    override suspend fun apps(token: String): ApiResult<List<AppInfo>> {
        appsCalls += 1
        lastToken = token
        return appsResult
    }

    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> =
        ApiResult.Success(PairResponse("tok"))

    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))
}

private class FakeTokenStore(initial: Map<String, String> = emptyMap()) : TokenStore {
    private val map = initial.toMutableMap()
    private val pins = mutableMapOf<String, String>()
    override fun getToken(serverUrl: String): String? = map[serverUrl]
    override fun setToken(serverUrl: String, token: String) {
        map[serverUrl] = token
    }
    override fun clearToken(serverUrl: String) {
        map.remove(serverUrl)
    }
    override fun getPin(serverUrl: String): String? = pins[serverUrl]
    override fun setPin(serverUrl: String, pinHex: String) { pins[serverUrl] = pinHex }
    override fun clearPin(serverUrl: String) { pins.remove(serverUrl) }
}

@OptIn(ExperimentalCoroutinesApi::class)
class AppsViewModelTest {

    private val testDispatcher = StandardTestDispatcher()
    private val serverUrl = "https://example.com:8443"

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun vm(
        api: FakeApi = FakeApi(),
        store: FakeTokenStore = FakeTokenStore(mapOf(serverUrl to "token-1")),
    ): Pair<AppsViewModel, FakeApi> {
        val repo = AppsRepository(
            apiFactory = { api },
            tokenStore = store,
            serverUrl = serverUrl,
        )
        return AppsViewModel(repo) to api
    }

    @Test
    fun initial_state_is_loading_before_dispatch() = runTest(testDispatcher) {
        val (model, _) = vm()
        assertEquals(AppsUiState.Loading, model.state.value)
        advanceUntilIdle()
    }

    @Test
    fun success_with_apps_moves_to_loaded() = runTest(testDispatcher) {
        val apps = listOf(
            AppInfo(id = "firefox", name = "Firefox"),
            AppInfo(id = "obsidian", name = "Obsidian"),
        )
        val (model, api) = vm(api = FakeApi(appsResult = ApiResult.Success(apps)))
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is AppsUiState.Loaded)
        assertEquals(apps, (s as AppsUiState.Loaded).apps)
        assertEquals("token-1", api.lastToken)
        assertEquals(1, api.appsCalls)
    }

    @Test
    fun success_with_empty_list_moves_to_empty() = runTest(testDispatcher) {
        val (model, _) = vm(api = FakeApi(appsResult = ApiResult.Success(emptyList())))
        advanceUntilIdle()
        assertEquals(AppsUiState.Empty, model.state.value)
    }

    @Test
    fun missing_token_moves_to_unauthorized() = runTest(testDispatcher) {
        val (model, api) = vm(store = FakeTokenStore(emptyMap()))
        advanceUntilIdle()
        assertEquals(AppsUiState.Unauthorized, model.state.value)
        assertEquals("repo should not call api without token", 0, api.appsCalls)
    }

    @Test
    fun http_401_moves_to_unauthorized() = runTest(testDispatcher) {
        val (model, _) = vm(api = FakeApi(appsResult = ApiResult.HttpError(401)))
        advanceUntilIdle()
        assertEquals(AppsUiState.Unauthorized, model.state.value)
    }

    @Test
    fun http_500_moves_to_error() = runTest(testDispatcher) {
        val (model, _) = vm(api = FakeApi(appsResult = ApiResult.HttpError(500)))
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is AppsUiState.Error)
        assertTrue((s as AppsUiState.Error).message.contains("500"))
    }

    @Test
    fun network_error_moves_to_error() = runTest(testDispatcher) {
        val (model, _) = vm(
            api = FakeApi(appsResult = ApiResult.NetworkError(RuntimeException("boom")))
        )
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is AppsUiState.Error)
        assertTrue((s as AppsUiState.Error).message.contains("boom"))
    }

    @Test
    fun refresh_resets_to_loading_before_request() = runTest(testDispatcher) {
        val api = FakeApi(appsResult = ApiResult.Success(listOf(AppInfo("a", "A"))))
        val (model, _) = vm(api = api)
        advanceUntilIdle()
        assertTrue(model.state.value is AppsUiState.Loaded)

        api.appsResult = ApiResult.Success(listOf(AppInfo("b", "B")))
        model.refresh()
        assertEquals(AppsUiState.Loading, model.state.value)

        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is AppsUiState.Loaded)
        assertEquals("B", (s as AppsUiState.Loaded).apps.single().name)
    }
}
