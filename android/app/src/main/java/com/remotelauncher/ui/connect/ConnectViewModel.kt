package com.remotelauncher.ui.connect

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.PinHolder
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.net.ServerStatus
import com.remotelauncher.net.SpkiPinCalculator
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
    data class PinConfirmRequired(
        val serverUrl: String,
        val pinHex: String,
        val displayFingerprint: String,
        val status: ServerStatus,
    ) : ConnectUiState()
    data class Connected(val serverUrl: String, val status: ServerStatus) : ConnectUiState()
    data class ConnectionFailed(val message: String) : ConnectUiState()
}

class ConnectViewModel(
    private val repo: SettingsRepository,
    private val apiFactory: (String) -> RemoteLauncherApi,
    private val tokenStore: TokenStore,
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
                    runConnect(parsed.url)
                }
            }
        }
    }

    private suspend fun runConnect(serverUrl: String) {
        val savedPin = tokenStore.getPin(serverUrl)
        if (savedPin != null) {
            PinHolder.setCurrent(savedPin)
        } else {
            PinHolder.setCurrent(null)
            PinHolder.consumeObserved()
        }

        val api = apiFactory(serverUrl)
        when (val result = api.status()) {
            is ApiResult.Success -> {
                if (savedPin != null) {
                    repo.setServerUrl(serverUrl)
                    _state.value = ConnectUiState.Connected(serverUrl, result.value)
                    return
                }

                val observed = PinHolder.consumeObserved()
                if (observed == null) {
                    _state.value = ConnectUiState.ConnectionFailed(
                        "Не удалось получить сертификат сервера"
                    )
                    return
                }
                PinHolder.setCurrent(observed)

                val statusNormalized = result.value.certFingerprint.replace(":", "").uppercase()
                if (statusNormalized != observed.uppercase()) {
                    PinHolder.setCurrent(null)
                    _state.value = ConnectUiState.ConnectionFailed(
                        "Сервер прислал отпечаток, отличный от его настоящего сертификата. Возможно, MITM. Не подключайтесь."
                    )
                    return
                }

                _state.value = ConnectUiState.PinConfirmRequired(
                    serverUrl = serverUrl,
                    pinHex = observed,
                    displayFingerprint = SpkiPinCalculator.toFingerprint(observed),
                    status = result.value,
                )
            }
            is ApiResult.HttpError -> {
                _state.value = ConnectUiState.ConnectionFailed(
                    "Сервер ответил ${result.code}"
                )
            }
            is ApiResult.NetworkError -> {
                _state.value = ConnectUiState.ConnectionFailed(
                    "Нет сети: ${result.cause.message ?: result.cause.javaClass.simpleName}"
                )
            }
        }
    }

    fun confirmPin() {
        val s = _state.value as? ConnectUiState.PinConfirmRequired ?: return
        viewModelScope.launch {
            tokenStore.setPin(s.serverUrl, s.pinHex)
            PinHolder.setCurrent(s.pinHex)
            repo.setServerUrl(s.serverUrl)
            _state.value = ConnectUiState.Connected(s.serverUrl, s.status)
        }
    }

    fun dismissPin() {
        val s = _state.value
        if (s is ConnectUiState.PinConfirmRequired) {
            PinHolder.setCurrent(null)
        }
        _state.value = ConnectUiState.Idle
    }

    fun reset() {
        _state.value = ConnectUiState.Idle
    }

    class Factory(
        private val repo: SettingsRepository,
        private val apiFactory: (String) -> RemoteLauncherApi,
        private val tokenStore: TokenStore,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(ConnectViewModel::class.java))
            return ConnectViewModel(repo, apiFactory, tokenStore) as T
        }
    }
}
