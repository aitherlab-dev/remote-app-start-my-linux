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
import com.remotelauncher.data.AppsRepository
import com.remotelauncher.data.SettingsRepository
import com.remotelauncher.data.TokenStore
import com.remotelauncher.net.RemoteLauncherApi
import com.remotelauncher.ui.admin.AdminScreen
import com.remotelauncher.ui.apps.AppsScreen
import com.remotelauncher.ui.terminal.TerminalScreen
import com.remotelauncher.ui.apps.AppsViewModel
import com.remotelauncher.ui.connect.ConnectScreen
import com.remotelauncher.ui.connect.ConnectViewModel
import com.remotelauncher.ui.pairing.PairingScreen
import com.remotelauncher.ui.pairing.PairingViewModel

object Routes {
    const val CONNECT = "connect"
    const val PAIRING = "pairing/{serverUrl}"
    const val APPS = "apps/{serverUrl}"
    const val ADMIN = "admin/{serverUrl}"
    const val TERMINAL = "terminal/{serverUrl}"

    fun pairing(serverUrl: String): String =
        "pairing/" + Uri.encode(serverUrl)

    fun apps(serverUrl: String): String =
        "apps/" + Uri.encode(serverUrl)

    fun admin(serverUrl: String): String =
        "admin/" + Uri.encode(serverUrl)

    fun terminal(serverUrl: String): String =
        "terminal/" + Uri.encode(serverUrl)
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
                factory = ConnectViewModel.Factory(settingsRepository, apiFactory, tokenStore)
            )
            ConnectScreen(
                viewModel = vm,
                onConnected = { url ->
                    val target = if (tokenStore.hasToken(url)) {
                        Routes.apps(url)
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
                    navController.navigate(Routes.apps(serverUrl)) {
                        popUpTo(Routes.CONNECT) { inclusive = true }
                    }
                },
            )
        }
        composable(
            route = Routes.APPS,
            arguments = listOf(navArgument("serverUrl") { type = NavType.StringType }),
        ) { backStackEntry ->
            val encoded = backStackEntry.arguments?.getString("serverUrl").orEmpty()
            val serverUrl = Uri.decode(encoded)
            val repo = AppsRepository(
                apiFactory = apiFactory,
                tokenStore = tokenStore,
                serverUrl = serverUrl,
            )
            val vm: AppsViewModel = viewModel(
                key = "apps-$serverUrl",
                factory = AppsViewModel.Factory(repo),
            )
            val authToken = tokenStore.getToken(serverUrl).orEmpty()
            AppsScreen(
                viewModel = vm,
                serverUrl = serverUrl,
                authToken = authToken,
                onUnauthorized = {
                    tokenStore.clearToken(serverUrl)
                    navController.navigate(Routes.pairing(serverUrl)) {
                        popUpTo(Routes.CONNECT) { inclusive = false }
                    }
                },
                onDisconnect = {
                    tokenStore.clearToken(serverUrl)
                    navController.navigate(Routes.CONNECT) {
                        popUpTo(Routes.CONNECT) { inclusive = true }
                    }
                },
                onAdmin = {
                    navController.navigate(Routes.admin(serverUrl))
                },
                onTerminal = {
                    navController.navigate(Routes.terminal(serverUrl))
                },
            )
        }
        composable(
            route = Routes.ADMIN,
            arguments = listOf(navArgument("serverUrl") { type = NavType.StringType }),
        ) { backStackEntry ->
            val encoded = backStackEntry.arguments?.getString("serverUrl").orEmpty()
            val serverUrl = Uri.decode(encoded)
            val authToken = tokenStore.getToken(serverUrl).orEmpty()
            AdminScreen(
                serverUrl = serverUrl,
                authToken = authToken,
                onBack = { navController.popBackStack() },
            )
        }
        composable(
            route = Routes.TERMINAL,
            arguments = listOf(navArgument("serverUrl") { type = NavType.StringType }),
        ) { backStackEntry ->
            val encoded = backStackEntry.arguments?.getString("serverUrl").orEmpty()
            val serverUrl = Uri.decode(encoded)
            val authToken = tokenStore.getToken(serverUrl).orEmpty()
            TerminalScreen(
                serverUrl = serverUrl,
                authToken = authToken,
                onBack = { navController.popBackStack() },
            )
        }
    }
}
