package com.remotelauncher.ui.connect

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.ServerStatus
import com.remotelauncher.util.ParsedUrl
import com.remotelauncher.util.parseServerUrl
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

sealed class ConnectUiState {
    data object Idle : ConnectUiState()
    data class InputError(val message: String) : ConnectUiState()
    data object Connecting : ConnectUiState()
    data class Connected(val serverUrl: String, val status: ServerStatus) : ConnectUiState()
    data class ConnectionFailed(val message: String) : ConnectUiState()
}

class ConnectViewModel(
    private val repo: SettingsRepository,
    private val apiFactory: (String) -> RemoteLauncherApi,
) : ViewModel() {

    private val _state = MutableStateFlow<ConnectUiState>(ConnectUiState.Idle)
    val state: StateFlow<ConnectUiState> = _state.asStateFlow()

    val savedUrl: StateFlow<String> = repo.serverUrl
        .map { it.orEmpty() }
        .stateIn(
            scope = viewModelScope,
            started = SharingStarted.Eagerly,
            initialValue = ""
        )

    fun connect(rawInput: String) {
        when (val parsed = parseServerUrl(rawInput)) {
            is ParsedUrl.Invalid -> {
                _state.value = ConnectUiState.InputError(parsed.reason)
            }
            is ParsedUrl.Valid -> {
                _state.value = ConnectUiState.Connecting
                viewModelScope.launch {
                    val api = apiFactory(parsed.url)
                    when (val result = api.status()) {
                        is ApiResult.Success -> {
                            repo.setServerUrl(parsed.url)
                            _state.value = ConnectUiState.Connected(parsed.url, result.value)
                        }
                        is ApiResult.HttpError -> {
                            _state.value = ConnectUiState.ConnectionFailed(
                                "Сервер ответил ${result.code}"
                            )
                        }
                        is ApiResult.NetworkError -> {
                            _state.value = ConnectUiState.ConnectionFailed(
                                "Нет сети: ${result.cause.message ?: ""}"
                            )
                        }
                    }
                }
            }
        }
    }

    fun reset() {
        _state.value = ConnectUiState.Idle
    }

    class Factory(
        private val repo: SettingsRepository,
        private val apiFactory: (String) -> RemoteLauncherApi,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(ConnectViewModel::class.java))
            return ConnectViewModel(repo, apiFactory) as T
        }
    }
}
