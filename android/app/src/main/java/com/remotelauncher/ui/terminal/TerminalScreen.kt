package com.remotelauncher.ui.terminal

import android.annotation.SuppressLint
import android.webkit.WebView
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView

private val KeyBg = Color(0xFF313244)
private val KeyBgMod = Color(0xFF45475A)
private val KeyBgModOn = Color(0xFF89B4FA)
private val KeyFg = Color(0xFFCDD6F4)
private val KeyFgModOn = Color(0xFF1E1E2E)
private val BarBg = Color(0xFF181825)

private data class TermKey(val label: String, val seq: String)

private val KEYS = listOf(
    TermKey("Esc", "\u001b"),
    TermKey("Tab", "\t"),
    TermKey("↑", "\u001b[A"),
    TermKey("↓", "\u001b[B"),
    TermKey("←", "\u001b[D"),
    TermKey("→", "\u001b[C"),
    TermKey("-", "-"),
    TermKey("|", "|"),
    TermKey("/", "/"),
    TermKey("~", "~"),
)

@OptIn(ExperimentalMaterial3Api::class)
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TerminalScreen(
    serverUrl: String,
    authToken: String,
    onBack: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val bridge = remember { arrayOfNulls<TerminalBridge>(1) }
    var ctrlOn by remember { mutableStateOf(false) }
    var altOn by remember { mutableStateOf(false) }

    DisposableEffect(Unit) {
        onDispose { bridge[0]?.disconnect() }
    }

    fun sendSeq(seq: String) {
        var s = seq
        if (altOn) { s = "\u001b$s"; altOn = false }
        if (ctrlOn && s.length == 1) {
            s = (s.uppercase()[0].code and 0x1f).toChar().toString()
            ctrlOn = false
        }
        bridge[0]?.sendInputFromUI(s)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Терминал") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Назад",
                        )
                    }
                },
            )
        },
        bottomBar = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .imePadding()
                    .background(BarBg)
                    .padding(3.dp),
                verticalArrangement = Arrangement.spacedBy(3.dp),
            ) {
                // Row 1: Esc | Alt | / | ↑ | ↓
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(3.dp),
                ) {
                    TermButton("Esc", KeyBg, KeyFg, { sendSeq("\u001b") }, Modifier.weight(1f))
                    TermButton("Alt",
                        if (altOn) KeyBgModOn else KeyBgMod,
                        if (altOn) KeyFgModOn else KeyFg,
                        { altOn = !altOn }, Modifier.weight(1f))
                    TermButton("/", KeyBg, KeyFg, { sendSeq("/") }, Modifier.weight(1f))
                    TermButton("↑", KeyBg, KeyFg, { sendSeq("\u001b[A") }, Modifier.weight(1f))
                    TermButton("↓", KeyBg, KeyFg, { sendSeq("\u001b[B") }, Modifier.weight(1f))
                }
                // Row 2: Tab | Ctrl | - | → | ←
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(3.dp),
                ) {
                    TermButton("Tab", KeyBg, KeyFg, { sendSeq("\t") }, Modifier.weight(1f))
                    TermButton("Ctrl",
                        if (ctrlOn) KeyBgModOn else KeyBgMod,
                        if (ctrlOn) KeyFgModOn else KeyFg,
                        { ctrlOn = !ctrlOn }, Modifier.weight(1f))
                    TermButton("-", KeyBg, KeyFg, { sendSeq("-") }, Modifier.weight(1f))
                    TermButton("→", KeyBg, KeyFg, { sendSeq("\u001b[C") }, Modifier.weight(1f))
                    TermButton("←", KeyBg, KeyFg, { sendSeq("\u001b[D") }, Modifier.weight(1f))
                }
            }
        },
    ) { padding ->
        AndroidView(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            factory = { context ->
                WebView(context).apply {
                    settings.javaScriptEnabled = true
                    settings.domStorageEnabled = true

                    val b = TerminalBridge(this, context, serverUrl, authToken, scope)
                    bridge[0] = b
                    addJavascriptInterface(b, "AndroidBridge")

                    loadUrl("file:///android_asset/terminal.html")
                }
            },
        )
    }
}

@Composable
private fun TermButton(
    label: String,
    bg: Color,
    fg: Color,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Text(
        text = label,
        color = fg,
        fontSize = 14.sp,
        fontFamily = FontFamily.Monospace,
        maxLines = 1,
        modifier = modifier
            .height(42.dp)
            .clip(RoundedCornerShape(5.dp))
            .background(bg)
            .clickable(onClick = onClick)
            .padding(horizontal = 8.dp, vertical = 10.dp),
        textAlign = TextAlign.Center,
    )
}
