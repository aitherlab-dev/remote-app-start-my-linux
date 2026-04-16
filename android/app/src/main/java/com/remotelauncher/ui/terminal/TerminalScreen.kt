package com.remotelauncher.ui.terminal

import androidx.compose.runtime.Composable
import com.remotelauncher.ui.webview.WebViewScreen

@Composable
fun TerminalScreen(
    serverUrl: String,
    authToken: String,
    onBack: () -> Unit,
) {
    WebViewScreen(
        url = "$serverUrl/terminal/",
        authToken = authToken,
        title = "Терминал",
        onBack = onBack,
    )
}
