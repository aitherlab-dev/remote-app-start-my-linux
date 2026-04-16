package com.remotelauncher.ui.webview

import android.net.http.SslError
import android.webkit.SslErrorHandler
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import com.remotelauncher.net.PinHolder
import com.remotelauncher.net.SpkiPinCalculator
import java.io.ByteArrayInputStream
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate

/**
 * WebViewClient that validates the server's TLS certificate against the
 * SPKI pin stored in [PinHolder]. This lets the WebView load pages from
 * the self-signed RemoteLauncher server without blanket-accepting all
 * SSL errors.
 */
class PinnedWebViewClient : WebViewClient() {

    override fun onReceivedSslError(
        view: WebView?,
        handler: SslErrorHandler?,
        error: SslError?,
    ) {
        val cert = error?.certificate
        if (cert == null) {
            handler?.cancel()
            return
        }

        // SslCertificate doesn't expose the public key directly.
        // Convert it to an X509Certificate via the DER encoding
        // accessible through the (deprecated but still functional)
        // SslCertificate.saveState bundle trick, or the newer
        // X509Certificate field.
        val x509 = try {
            val bundle = android.net.http.SslCertificate.saveState(cert)
            val derBytes = bundle.getByteArray("x509-certificate")
            if (derBytes != null) {
                CertificateFactory.getInstance("X.509")
                    .generateCertificate(ByteArrayInputStream(derBytes)) as X509Certificate
            } else {
                null
            }
        } catch (_: Exception) {
            null
        }

        if (x509 == null) {
            handler?.cancel()
            return
        }

        val observedHex = SpkiPinCalculator.computeHex(x509)
        val expected = PinHolder.getCurrent()

        if (expected != null && expected.equals(observedHex, ignoreCase = true)) {
            handler?.proceed()
        } else {
            handler?.cancel()
        }
    }

    override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
        val url = request?.url ?: return false
        val pageHost = view?.url?.let { android.net.Uri.parse(it).host }
        // Keep same-origin navigation inside the WebView.
        return url.host != pageHost
    }
}
