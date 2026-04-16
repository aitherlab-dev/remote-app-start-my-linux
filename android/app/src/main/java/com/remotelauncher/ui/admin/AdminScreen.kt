package com.remotelauncher.ui.admin

import androidx.compose.runtime.Composable
import com.remotelauncher.ui.webview.WebViewScreen

@Composable
fun AdminScreen(
    serverUrl: String,
    authToken: String,
    onBack: () -> Unit,
) {
    WebViewScreen(
        url = "$serverUrl/admin/",
        authToken = authToken,
        title = "Настройки",
        onBack = onBack,
    )
}
