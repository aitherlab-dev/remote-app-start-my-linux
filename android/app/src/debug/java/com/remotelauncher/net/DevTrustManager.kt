package com.remotelauncher.net

import java.security.cert.X509Certificate
import javax.net.ssl.X509TrustManager

// TODO(A5.1): DELETE THIS FILE, replace with PinnedTrustManager.
//
// TEMPORARY dev-only TrustManager that accepts ANY certificate chain,
// including self-signed. Exists ONLY to unblock development of screens
// A3/A4 against a real local server before proper SPKI pinning lands
// in A5.1.
//
// WARNING: using this in a release build is a security hole. This file
// lives under src/debug/java/, so it physically cannot be part of the
// release APK — the debug variant of HttpClientFactory is the only thing
// that references it.
internal class DevTrustManager : X509TrustManager {
    override fun checkClientTrusted(chain: Array<out X509Certificate>?, authType: String?) {
        // noop — dev only
    }

    override fun checkServerTrusted(chain: Array<out X509Certificate>?, authType: String?) {
        // noop — dev only
    }

    override fun getAcceptedIssuers(): Array<X509Certificate> = emptyArray()
}
