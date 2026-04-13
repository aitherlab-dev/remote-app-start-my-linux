package com.remotelauncher.ui.connect

import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import com.remotelauncher.net.LaunchResponse
import com.remotelauncher.net.PairResponse
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
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

private class FakeRemoteLauncherApi(
    var nextStatus: ApiResult<ServerStatus> = ApiResult.Success(SAMPLE_STATUS),
) : RemoteLauncherApi {
    var statusCalls: Int = 0
        private set

    override suspend fun status(): ApiResult<ServerStatus> {
        statusCalls += 1
        return nextStatus
    }

    override suspend fun apps(token: String): ApiResult<List<AppInfo>> =
        ApiResult.Success(emptyList())

    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> =
        ApiResult.Success(PairResponse("token"))

    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> =
        ApiResult.Success(LaunchResponse("ok", 1))

    companion object {
        val SAMPLE_STATUS = ServerStatus(
            version = "0.1.0",
            startedAt = "2026-04-13T12:00:00Z",
            uptimeSec = 42,
            appsCount = 17,
            certFingerprint = "deadbeef",
        )
    }
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
    }

    @After
    fun tearDown() {
        storeScope.cancel()
        Dispatchers.resetMain()
    }

    private fun newRepository(fileName: String = "settings.preferences_pb"): SettingsRepository {
        val file = File(tempFolder.newFolder(), fileName)
        val store = PreferenceDataStoreFactory.create(scope = storeScope) { file }
        return SettingsRepository(store)
    }

    @Test
    fun initial_state_is_idle() = runTest(testDispatcher) {
        val repo = newRepository()
        val vm = ConnectViewModel(repo) { FakeRemoteLauncherApi() }
        advanceUntilIdle()
        assertEquals(ConnectUiState.Idle, vm.state.value)
    }

    @Test
    fun invalid_input_shows_error() = runTest(testDispatcher) {
        val repo = newRepository()
        val vm = ConnectViewModel(repo) { FakeRemoteLauncherApi() }
        vm.connect("   ")
        advanceUntilIdle()
        assertTrue(vm.state.value is ConnectUiState.InputError)
    }

    @Test
    fun valid_input_then_success_saves_url_and_shows_connected() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi()
        var factoryArg: String? = null
        val vm = ConnectViewModel(repo) { url ->
            factoryArg = url
            fake
        }
        vm.connect("localhost:8443")
        advanceUntilIdle()
        val state = vm.state.value
        assertTrue("state was $state", state is ConnectUiState.Connected)
        assertEquals("https://localhost:8443", factoryArg)
        assertEquals("https://localhost:8443", repo.serverUrl.first())
        assertEquals(1, fake.statusCalls)
    }

    @Test
    fun http_error_shows_connection_failed() = runTest(testDispatcher) {
        val repo = newRepository()
        val fake = FakeRemoteLauncherApi(nextStatus = ApiResult.HttpError(500, "bad"))
        val vm = ConnectViewModel(repo) { fake }
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
            nextStatus = ApiResult.NetworkError(RuntimeException("boom"))
        )
        val vm = ConnectViewModel(repo) { fake }
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
        val vm = ConnectViewModel(repo) { FakeRemoteLauncherApi() }
        advanceUntilIdle()
        assertEquals("https://preset.example:8443", vm.savedUrl.value)
    }
}
