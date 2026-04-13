package com.remotelauncher.ui.apps

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.remotelauncher.data.AppsRepository
import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed class AppsUiState {
    data object Loading : AppsUiState()
    data class Loaded(val apps: List<AppInfo>) : AppsUiState()
    data object Empty : AppsUiState()
    data class Error(val message: String) : AppsUiState()
    data object Unauthorized : AppsUiState()
}

class AppsViewModel(
    private val repo: AppsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow<AppsUiState>(AppsUiState.Loading)
    val state: StateFlow<AppsUiState> = _state.asStateFlow()

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
