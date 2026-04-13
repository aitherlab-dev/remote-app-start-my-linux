package com.remotelauncher.ui.pairing

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.RemoteLauncherApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed class PairingUiState {
    data object EnterPin : PairingUiState()
    data object Sending : PairingUiState()
    data object Paired : PairingUiState()
    data class Error(val message: String) : PairingUiState()
}

class PairingViewModel(
    private val serverUrl: String,
    private val apiFactory: (String) -> RemoteLauncherApi,
    private val tokenStore: TokenStore,
    private val deviceLabel: String,
) : ViewModel() {

    private val _state = MutableStateFlow<PairingUiState>(PairingUiState.EnterPin)
    val state: StateFlow<PairingUiState> = _state.asStateFlow()

    fun submit(pin: String) {
        if (pin.length != PIN_LENGTH || !pin.all { it.isDigit() }) {
            _state.value = PairingUiState.Error("PIN должен быть 6 цифр")
            return
        }
        _state.value = PairingUiState.Sending
        viewModelScope.launch {
            val api = apiFactory(serverUrl)
            when (val result = api.pair(pin, deviceLabel)) {
                is ApiResult.Success -> {
                    try {
                        tokenStore.setToken(serverUrl, result.value.token)
                        _state.value = PairingUiState.Paired
                    } catch (t: Throwable) {
                        _state.value = PairingUiState.Error(
                            "Не удалось сохранить токен: ${t.message ?: t.javaClass.simpleName}"
                        )
                    }
                }
                is ApiResult.HttpError -> {
                    val msg = when (result.code) {
                        401 -> "Неверный PIN"
                        429 -> "Слишком много попыток, подождите"
                        else -> "Сервер ответил ${result.code}"
                    }
                    _state.value = PairingUiState.Error(msg)
                }
                is ApiResult.NetworkError -> {
                    _state.value = PairingUiState.Error(
                        "Нет связи: ${result.cause.message ?: result.cause.javaClass.simpleName}"
                    )
                }
            }
        }
    }

    fun reset() {
        _state.value = PairingUiState.EnterPin
    }

    class Factory(
        private val serverUrl: String,
        private val apiFactory: (String) -> RemoteLauncherApi,
        private val tokenStore: TokenStore,
        private val deviceLabel: String,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(PairingViewModel::class.java))
            return PairingViewModel(serverUrl, apiFactory, tokenStore, deviceLabel) as T
        }
    }

    companion object {
        const val PIN_LENGTH = 6
    }
}
