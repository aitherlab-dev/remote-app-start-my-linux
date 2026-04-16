package com.remotelauncher.ui.terminal

import android.content.Context
import android.util.Base64
import android.util.Log
import android.webkit.JavascriptInterface
import android.webkit.WebView
import com.remotelauncher.net.PinnedTrustManager
import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.websocket.WebSockets
import io.ktor.client.plugins.websocket.wss
import io.ktor.websocket.Frame
import io.ktor.websocket.readBytes
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * JavaScript interface bridge that relays data between xterm.js in the
 * WebView and a Ktor WebSocket connection to the RemoteLauncher server.
 *
 * This approach bypasses the Android WebView limitation where
 * `new WebSocket()` from JS does not trust self-signed certificates
 * (even after `onReceivedSslError` → `handler.proceed()`).
 *
 * The Ktor CIO client uses [PinnedTrustManager] — the same SPKI
 * pinning as the rest of the app.
 */
class TerminalBridge(
    private val webView: WebView,
    private val context: Context,
    private val serverUrl: String,
    private val authToken: String,
    private val scope: CoroutineScope,
) {
    private var job: Job? = null
    private var sendFrame: (suspend (Frame) -> Unit)? = null

    @JavascriptInterface
    fun connect() {
        job?.cancel()
        job = scope.launch(Dispatchers.IO) {
            try {
                val client = HttpClient(CIO) {
                    engine {
                        https { trustManager = PinnedTrustManager() }
                    }
                    install(WebSockets)
                }
                val wsHost = serverUrl
                    .removePrefix("https://")
                    .removePrefix("http://")
                val hostParts = wsHost.split(":", limit = 2)
                val host = hostParts[0]
                val port = hostParts.getOrNull(1)?.toIntOrNull() ?: 8443

                client.wss(
                    host = host,
                    port = port,
                    path = "/terminal/ws?token=$authToken",
                ) {
                    sendFrame = { frame -> send(frame) }

                    // Notify JS that connection is open.
                    withContext(Dispatchers.Main) {
                        webView.evaluateJavascript("if(window._onOpen)_onOpen()", null)
                    }

                    for (frame in incoming) {
                        when (frame) {
                            is Frame.Binary, is Frame.Text -> {
                                val b64 = Base64.encodeToString(
                                    frame.readBytes(),
                                    Base64.NO_WRAP,
                                )
                                withContext(Dispatchers.Main) {
                                    webView.evaluateJavascript(
                                        "if(window._onData)_onData('$b64')",
                                        null,
                                    )
                                }
                            }
                            else -> {}
                        }
                    }
                }
            } catch (e: Exception) {
                Log.e("TerminalBridge", "WebSocket error", e)
                withContext(Dispatchers.Main) {
                    webView.evaluateJavascript(
                        "if(window._onError)_onError('${e.message?.replace("'", "\\'") ?: "unknown"}')",
                        null,
                    )
                }
            }
        }
    }

    @JavascriptInterface
    fun sendInput(b64: String) {
        val bytes = Base64.decode(b64, Base64.DEFAULT)
        scope.launch(Dispatchers.IO) {
            try {
                sendFrame?.invoke(Frame.Binary(true, bytes))
            } catch (_: Exception) {}
        }
    }

    @JavascriptInterface
    fun resize(cols: Int, rows: Int) {
        val json = """{"type":"resize","cols":$cols,"rows":$rows}"""
        scope.launch(Dispatchers.IO) {
            try {
                sendFrame?.invoke(Frame.Text(json))
            } catch (_: Exception) {}
        }
    }

    @JavascriptInterface
    fun onPageReady() {
        Log.d("TerminalBridge", "onPageReady — injecting xterm.js")
        scope.launch(Dispatchers.Main) {
            try {
                val xtermCss = context.assets.open("xterm.min.css")
                    .bufferedReader().readText()
                val xtermJs = context.assets.open("xterm.min.js")
                    .bufferedReader().readText()
                val fitJs = context.assets.open("addon-fit.min.js")
                    .bufferedReader().readText()

                // Inject CSS
                val escapedCss = xtermCss
                    .replace("\\", "\\\\")
                    .replace("`", "\\`")
                    .replace("\$", "\\$")
                webView.evaluateJavascript(
                    "(function(){var s=document.createElement('style');" +
                    "s.textContent=`$escapedCss`;" +
                    "document.head.appendChild(s)})()",
                    null,
                )

                // Inject xterm.js
                webView.evaluateJavascript(xtermJs, null)

                // Inject addon-fit, then boot
                webView.evaluateJavascript(fitJs) {
                    Log.d("TerminalBridge", "xterm injected, calling _boot")
                    webView.evaluateJavascript("if(window._boot)_boot()", null)
                }
            } catch (e: Exception) {
                Log.e("TerminalBridge", "inject failed", e)
                webView.evaluateJavascript(
                    "document.getElementById('status').textContent='inject error: ${e.message}'",
                    null,
                )
            }
        }
    }

    /** Called from native Compose key bar (not from JS). */
    fun sendInputFromUI(text: String) {
        val bytes = text.toByteArray(Charsets.UTF_8)
        scope.launch(Dispatchers.IO) {
            try {
                sendFrame?.invoke(Frame.Binary(true, bytes))
            } catch (_: Exception) {}
        }
    }

    fun disconnect() {
        job?.cancel()
        job = null
        sendFrame = null
    }
}
