package com.remotelauncher.ui

import androidx.compose.runtime.Composable
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.ui.connect.ConnectScreen
import com.remotelauncher.ui.connect.ConnectViewModel

object Routes {
    const val CONNECT = "connect"
}

@Composable
fun AppNavHost(
    settingsRepository: SettingsRepository,
    apiFactory: (String) -> RemoteLauncherApi,
    navController: NavHostController = rememberNavController(),
) {
    NavHost(
        navController = navController,
        startDestination = Routes.CONNECT,
    ) {
        composable(Routes.CONNECT) {
            val vm: ConnectViewModel = viewModel(
                factory = ConnectViewModel.Factory(settingsRepository, apiFactory)
            )
            ConnectScreen(viewModel = vm)
        }
    }
}
