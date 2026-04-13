package com.remotelauncher.ui.apps

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.remotelauncher.data.AppsRepository
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.launch

sealed class AppsUiState {
    data object Loading : AppsUiState()
    data class Loaded(val apps: List<AppInfo>) : AppsUiState()
    data object Empty : AppsUiState()
    data class Error(val message: String) : AppsUiState()
    data object Unauthorized : AppsUiState()
}

sealed class AppsUiEvent {
    data class Launching(val appName: String) : AppsUiEvent()
    data class Launched(val appName: String) : AppsUiEvent()
    data class LaunchFailed(val appName: String, val reason: String) : AppsUiEvent()
}

class AppsViewModel(
    private val repo: AppsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow<AppsUiState>(AppsUiState.Loading)
    val state: StateFlow<AppsUiState> = _state.asStateFlow()

    private val _events = Channel<AppsUiEvent>(capacity = Channel.BUFFERED)
    val events = _events.receiveAsFlow()

    @Volatile
    private var launchInFlight = false

    init {
        refresh()
    }

    fun refresh() {
        _state.value = AppsUiState.Loading
        viewModelScope.launch {
            _state.value = when (val r = repo.listApps()) {
                is ApiResult.Success -> if (r.value.isEmpty()) {
                    AppsUiState.Empty
                } else {
                    AppsUiState.Loaded(r.value)
                }
                is ApiResult.HttpError -> if (r.code == 401) {
                    AppsUiState.Unauthorized
                } else {
                    AppsUiState.Error("Сервер: ${r.code}")
                }
                is ApiResult.NetworkError -> AppsUiState.Error(
                    "Нет связи: ${r.cause.message ?: r.cause.javaClass.simpleName}"
                )
            }
        }
    }

    fun onTap(app: AppInfo) {
        if (launchInFlight) return
        launchInFlight = true
        viewModelScope.launch {
            _events.send(AppsUiEvent.Launching(app.name))
            when (val r = repo.launch(app.id)) {
                is ApiResult.Success -> _events.send(AppsUiEvent.Launched(app.name))
                is ApiResult.HttpError -> {
                    if (r.code == 401) {
                        _state.value = AppsUiState.Unauthorized
                    } else {
                        _events.send(AppsUiEvent.LaunchFailed(app.name, "сервер ${r.code}"))
                    }
                }
                is ApiResult.NetworkError -> _events.send(
                    AppsUiEvent.LaunchFailed(app.name, r.cause.message ?: r.cause.javaClass.simpleName)
                )
            }
            launchInFlight = false
        }
    }

    class Factory(
        private val repo: AppsRepository,
    ) : ViewModelProvider.Factory {
        @Suppress("UNCHECKED_CAST")
        override fun <T : ViewModel> create(modelClass: Class<T>): T {
            require(modelClass.isAssignableFrom(AppsViewModel::class.java))
            return AppsViewModel(repo) as T
        }
    }
}
