package com.remotelauncher.ui

import android.net.Uri
import androidx.compose.runtime.Composable
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.ui.apps.AppsScreen
import com.remotelauncher.ui.connect.ConnectScreen
import com.remotelauncher.ui.connect.ConnectViewModel
import com.remotelauncher.ui.pairing.PairingScreen
import com.remotelauncher.ui.pairing.PairingViewModel

object Routes {
    const val CONNECT = "connect"
    const val PAIRING = "pairing/{serverUrl}"
    const val APPS = "apps"

    fun pairing(serverUrl: String): String =
        "pairing/" + Uri.encode(serverUrl)
}

@Composable
fun AppNavHost(
    settingsRepository: SettingsRepository,
    apiFactory: (String) -> RemoteLauncherApi,
    tokenStore: TokenStore,
    deviceLabel: String,
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
            ConnectScreen(
                viewModel = vm,
                onConnected = { url ->
                    val target = if (tokenStore.hasToken(url)) {
                        Routes.APPS
                    } else {
                        Routes.pairing(url)
                    }
                    navController.navigate(target) {
                        popUpTo(Routes.CONNECT) { inclusive = true }
                    }
                },
            )
        }
        composable(
            route = Routes.PAIRING,
            arguments = listOf(navArgument("serverUrl") { type = NavType.StringType }),
        ) { backStackEntry ->
            val encoded = backStackEntry.arguments?.getString("serverUrl").orEmpty()
            val serverUrl = Uri.decode(encoded)
            val vm: PairingViewModel = viewModel(
                factory = PairingViewModel.Factory(
                    serverUrl = serverUrl,
                    apiFactory = apiFactory,
                    tokenStore = tokenStore,
                    deviceLabel = deviceLabel,
                )
            )
            PairingScreen(
                viewModel = vm,
                onPaired = {
                    navController.navigate(Routes.APPS) {
                        popUpTo(Routes.CONNECT) { inclusive = true }
                    }
                },
            )
        }
        composable(Routes.APPS) {
            AppsScreen()
        }
    }
}
