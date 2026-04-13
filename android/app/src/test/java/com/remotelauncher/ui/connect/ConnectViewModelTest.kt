package com.remotelauncher.ui.connect

import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import com.remotelauncher.net.LaunchResponse
import com.remotelauncher.net.PairResponse
import com.remotelauncher.net.PinHolder
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.ServerStatus
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

private class FakeRemoteLauncherApi(
    var nextStatus: ApiResult<ServerStatus> = ApiResult.Success(SAMPLE_STATUS),
    var simulatedObservedHex: String? = SAMPLE_STATUS.certFingerprint,
) : RemoteLauncherApi {
    var statusCalls: Int = 0
        private set

    override suspend fun status(): ApiResult<ServerStatus> {
        statusCalls += 1
        simulatedObservedHex?.let { PinHolder.recordObserved(it) }
        return nextStatus
    }

    override suspend fun apps(token: String): ApiResult<List<AppInfo>> =
        ApiResult.Success(emptyList())

    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> =
        ApiResult.Success(PairResponse("token"))

    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))

    companion object {
        const val SAMPLE_PIN_HEX = "DEADBEEF1234"
        val SAMPLE_STATUS = ServerStatus(
            version = "0.1.0",
            startedAt = "2026-04-13T12:00:00Z",
            uptimeSec = 42,
            appsCount = 17,
            certFingerprint = SAMPLE_PIN_HEX,
        )
    }
}

private class FakeTokenStore : TokenStore {
    private val tokens = mutableMapOf<String, String>()
    private val pins = mutableMapOf<String, String>()
    override fun getToken(serverUrl: String): String? = tokens[serverUrl]
    override fun setToken(serverUrl: String, token: String) { tokens[serverUrl] = token }
    override fun clearToken(serverUrl: String) { tokens.remove(serverUrl) }
    override fun getPin(serverUrl: String): String? = pins[serverUrl]
    override fun setPin(serverUrl: String, pinHex: String) { pins[serverUrl] = pinHex.uppercase() }
    override fun clearPin(serverUrl: String) { pins.remove(serverUrl) }
}

@OptIn(ExperimentalCoroutinesApi::class)
class ConnectViewModelTest {

    @get:Rule
    val tempFolder = TemporaryFolder()

    private val testDispatcher = StandardTestDispatcher()
    private lateinit var storeScope: CoroutineScope

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
        storeScope = CoroutineScope(SupervisorJob() + UnconfinedTestDispatcher(testDispatcher.scheduler))
        PinHolder.clear()
    }

    @After
    fun tearDown() {
        storeScope.cancel()
        Dispatchers.resetMain()
        PinHolder.clear()
    }

    private fun newRepository(fileName: String = "settings.preferences_pb"): SettingsRepository {
        val file = File(tempFolder.newFolder(), fileName)
        val store = PreferenceDataStoreFactory.create(scope = storeScope) { file }
        return SettingsRepository(store)
    }

    private fun newVm(
        repo: SettingsRepository,
        api: RemoteLauncherApi,
        tokenStore: TokenStore = FakeTokenStore(),
    ): ConnectViewModel = ConnectViewModel(repo, { api }, tokenStore)

    @Test
    fun initial_state_is_idle() = runTest(testDispatcher) {
        val repo = newRepository()
        val vm = newVm(repo, FakeRemoteLauncherApi())
        advanceUntilIdle()
        assertEquals(ConnectUiState.Idle, vm.state.value)
    }

    @Test
    fun invalid_input_shows_error() = runTest(testDispatcher) {
        val repo = newRepository()
        val vm = newVm(repo, FakeRemoteLauncherApi())
        vm.connect("   ")
        advanceUntilIdle()
        assertTrue(vm.state.value is ConnectUiState.InputError)
    }

    @Test
    fun bootstrap_then_success_shows_pin_confirm() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi()
        val vm = newVm(repo, fake)
        vm.connect("localhost:8443")
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.PinConfirmRequired)
        val s = state as ConnectUiState.PinConfirmRequired
        assertEquals("https://localhost:8443", s.serverUrl)
        assertEquals(FakeRemoteLauncherApi.SAMPLE_PIN_HEX, s.pinHex.uppercase())
        assertEquals(1, fake.statusCalls)
    }

    @Test
    fun confirmPin_persists_and_transitions_to_connected() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi()
        val store = FakeTokenStore()
        val vm = newVm(repo, fake, store)
        vm.connect("localhost:8443")
        advanceUntilIdle()
        vm.confirmPin()
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.Connected)
        assertEquals("https://localhost:8443", repo.serverUrl.first())
        assertEquals(
            FakeRemoteLauncherApi.SAMPLE_PIN_HEX.uppercase(),
            store.getPin("https://localhost:8443"),
        )
        assertEquals(FakeRemoteLauncherApi.SAMPLE_PIN_HEX.uppercase(), PinHolder.getCurrent())
    }

    @Test
    fun saved_pin_skips_dialog_and_goes_connected() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi()
        val store = FakeTokenStore()
        store.setPin("https://localhost:8443", FakeRemoteLauncherApi.SAMPLE_PIN_HEX)
        val vm = newVm(repo, fake, store)
        vm.connect("localhost:8443")
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.Connected)
        assertEquals("https://localhost:8443", repo.serverUrl.first())
    }

    @Test
    fun status_fingerprint_mismatch_fails_bootstrap() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi(
            simulatedObservedHex = "AABBCCDD"
        )
        val vm = newVm(repo, fake)
        vm.connect("localhost:8443")
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.ConnectionFailed)
        assertNull(repo.serverUrl.first())
        assertNull(PinHolder.getCurrent())
    }

    @Test
    fun dismissPin_clears_pin_holder_and_resets() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi()
        val store = FakeTokenStore()
        val vm = newVm(repo, fake, store)
        vm.connect("localhost:8443")
        advanceUntilIdle()
        vm.dismissPin()
        advanceUntilIdle()
        assertEquals(ConnectUiState.Idle, vm.state.value)
        assertNull(store.getPin("https://localhost:8443"))
        assertNull(PinHolder.getCurrent())
    }

    @Test
    fun http_error_shows_connection_failed() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi(nextStatus = ApiResult.HttpError(500, "bad"))
        val vm = newVm(repo, fake)
        vm.connect("localhost")
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.ConnectionFailed)
        assertTrue((state as ConnectUiState.ConnectionFailed).message.contains("500"))
        assertEquals(null, repo.serverUrl.first())
    }

    @Test
    fun network_error_shows_connection_failed() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi(
            nextStatus = ApiResult.NetworkError(RuntimeException("boom")),
            simulatedObservedHex = null,
        )
        val vm = newVm(repo, fake)
        vm.connect("localhost")
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.ConnectionFailed)
        assertTrue((state as ConnectUiState.ConnectionFailed).message.contains("boom"))
    }

    @Test
    fun savedUrl_is_loaded_on_init() = runTest(testDispatcher) {
        val repo = newRepository()
        repo.setServerUrl("https://preset.example:8443")
        advanceUntilIdle()
        val vm = newVm(repo, FakeRemoteLauncherApi())
        advanceUntilIdle()
        assertEquals("https://preset.example:8443", vm.savedUrl.value)
        assertNotNull(vm)
    }
}
