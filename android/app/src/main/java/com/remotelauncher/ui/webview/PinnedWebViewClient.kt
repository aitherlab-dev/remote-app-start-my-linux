package com.remotelauncher.ui.webview

import android.net.http.SslError
import android.os.Build
import android.util.Log
import android.webkit.SslErrorHandler
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import com.remotelauncher.net.PinHolder
import com.remotelauncher.net.SpkiPinCalculator
import java.io.ByteArrayInputStream
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate

class PinnedWebViewClient : WebViewClient() {

    override fun onReceivedSslError(
        view: WebView?,
        handler: SslErrorHandler?,
        error: SslError?,
    ) {
        Log.d(TAG, "onReceivedSslError: type=${error?.primaryError}, url=${error?.url}")

        val cert = error?.certificate
        if (cert == null) {
            Log.w(TAG, "no certificate in SslError")
            handler?.cancel()
            return
        }

        val x509 = extractX509(cert)
        if (x509 == null) {
            Log.w(TAG, "failed to extract X509Certificate")
            handler?.cancel()
            return
        }

        val observedHex = SpkiPinCalculator.computeHex(x509)
        val expected = PinHolder.getCurrent()
        Log.d(TAG, "pin check: expected=${expected?.take(16)}… observed=${observedHex.take(16)}…")

        if (expected != null && expected.equals(observedHex, ignoreCase = true)) {
            Log.d(TAG, "pin matched — proceeding")
            handler?.proceed()
        } else {
            Log.w(TAG, "pin mismatch or no pin — cancelling")
            handler?.cancel()
        }
    }

    override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
        val url = request?.url ?: return false
        val pageHost = view?.url?.let { android.net.Uri.parse(it).host }
        return url.host != pageHost
    }

    private fun extractX509(cert: android.net.http.SslCertificate): X509Certificate? {
        // API 29+: direct accessor
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            try {
                val x509 = cert.x509Certificate
                if (x509 != null) {
                    Log.d(TAG, "extracted via getX509Certificate()")
                    return x509
                }
            } catch (e: Exception) {
                Log.w(TAG, "getX509Certificate() failed: ${e.message}")
            }
        }

        // Fallback: saveState bundle
        try {
            val bundle = android.net.http.SslCertificate.saveState(cert)
            val derBytes = bundle.getByteArray("x509-certificate")
            if (derBytes != null) {
                Log.d(TAG, "extracted via saveState bundle (${derBytes.size} bytes)")
                return CertificateFactory.getInstance("X.509")
                    .generateCertificate(ByteArrayInputStream(derBytes)) as X509Certificate
            }
            Log.w(TAG, "saveState bundle has no x509-certificate key")
        } catch (e: Exception) {
            Log.w(TAG, "saveState fallback failed: ${e.message}")
        }

        return null
    }

    companion object {
        private const val TAG = "PinnedWVC"
    }
}
