package com.remotelauncher.ui.pairing

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
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

private class FakeApi(
    var pairResult: ApiResult<PairResponse> = ApiResult.Success(PairResponse("token-abc")),
) : RemoteLauncherApi {
    var pairCalls: Int = 0
        private set
    var lastPin: String? = null
        private set
    var lastDeviceLabel: String? = null
        private set

    override suspend fun status(): ApiResult<ServerStatus> =
        ApiResult.NetworkError(RuntimeException("not used"))

    override suspend fun apps(token: String): ApiResult<List<AppInfo>> =
        ApiResult.Success(emptyList())

    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> {
        pairCalls += 1
        lastPin = pin
        lastDeviceLabel = deviceLabel
        return pairResult
    }

    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))
}

private class FakeTokenStore(
    private val throwOnSet: Boolean = false,
) : TokenStore {
    private val map = mutableMapOf<String, String>()
    private val pins = mutableMapOf<String, String>()
    var setCalls: Int = 0
        private set

    override fun getToken(serverUrl: String): String? = map[serverUrl]

    override fun setToken(serverUrl: String, token: String) {
        setCalls += 1
        if (throwOnSet) throw RuntimeException("keystore boom")
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
class PairingViewModelTest {

    private val testDispatcher = StandardTestDispatcher()
    private val serverUrl = "https://example.com:8443"
    private val deviceLabel = "samsung SM-S938B"

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
        store: FakeTokenStore = FakeTokenStore(),
    ): Pair<PairingViewModel, Pair<FakeApi, FakeTokenStore>> {
        val model = PairingViewModel(
            serverUrl = serverUrl,
            apiFactory = { api },
            tokenStore = store,
            deviceLabel = deviceLabel,
        )
        return model to (api to store)
    }

    @Test
    fun initial_state_is_enterPin() = runTest(testDispatcher) {
        val (model, _) = vm()
        assertEquals(PairingUiState.EnterPin, model.state.value)
    }

    @Test
    fun shortPin_showsError_andSkipsApi() = runTest(testDispatcher) {
        val (model, deps) = vm()
        model.submit("123")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertEquals(0, deps.first.pairCalls)
    }

    @Test
    fun nonDigits_showsError_andSkipsApi() = runTest(testDispatcher) {
        val (model, deps) = vm()
        model.submit("abcdef")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertEquals(0, deps.first.pairCalls)
    }

    @Test
    fun happyPath_savesToken_andMovesToPaired() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.Success(PairResponse("token-123")))
        val store = FakeTokenStore()
        val (model, _) = vm(api, store)
        model.submit("654321")
        advanceUntilIdle()
        assertEquals(PairingUiState.Paired, model.state.value)
        assertEquals("token-123", store.getToken(serverUrl))
        assertEquals(1, api.pairCalls)
        assertEquals("654321", api.lastPin)
        assertEquals(deviceLabel, api.lastDeviceLabel)
    }

    @Test
    fun http401_showsWrongPinMessage() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.HttpError(401))
        val store = FakeTokenStore()
        val (model, _) = vm(api, store)
        model.submit("111111")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertEquals("Неверный PIN", (s as PairingUiState.Error).message)
        assertNull(store.getToken(serverUrl))
    }

    @Test
    fun http429_showsRateLimitMessage() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.HttpError(429))
        val (model, _) = vm(api)
        model.submit("222222")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertTrue((s as PairingUiState.Error).message.contains("Слишком много"))
    }

    @Test
    fun http500_showsGenericMessage() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.HttpError(500))
        val (model, _) = vm(api)
        model.submit("333333")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertTrue((s as PairingUiState.Error).message.contains("500"))
    }

    @Test
    fun networkError_showsErrorMessage() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.NetworkError(RuntimeException("boom")))
        val (model, _) = vm(api)
        model.submit("444444")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertTrue((s as PairingUiState.Error).message.contains("boom"))
    }

    @Test
    fun tokenStoreThrow_showsError_andNotPaired() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.Success(PairResponse("token-xyz")))
        val store = FakeTokenStore(throwOnSet = true)
        val (model, _) = vm(api, store)
        model.submit("555555")
        advanceUntilIdle()
        val s = model.state.value
        assertTrue("state was $s", s is PairingUiState.Error)
        assertTrue((s as PairingUiState.Error).message.contains("Не удалось"))
        assertNull(store.getToken(serverUrl))
    }

    @Test
    fun reset_returnsToEnterPin() = runTest(testDispatcher) {
        val api = FakeApi(pairResult = ApiResult.HttpError(401))
        val (model, _) = vm(api)
        model.submit("111111")
        advanceUntilIdle()
        assertTrue(model.state.value is PairingUiState.Error)
        model.reset()
        assertEquals(PairingUiState.EnterPin, model.state.value)
    }
}
