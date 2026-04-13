package com.remotelauncher.net

import java.net.Socket
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import javax.net.ssl.SSLEngine
import javax.net.ssl.X509ExtendedTrustManager

internal class PinnedTrustManager : X509ExtendedTrustManager() {

    override fun checkClientTrusted(chain: Array<out X509Certificate>?, authType: String?) = Unit

    override fun checkServerTrusted(chain: Array<out X509Certificate>?, authType: String?) {
        verify(chain)
    }

    override fun checkClientTrusted(
        chain: Array<out X509Certificate>?,
        authType: String?,
        socket: Socket?,
    ) = Unit

    override fun checkServerTrusted(
        chain: Array<out X509Certificate>?,
        authType: String?,
        socket: Socket?,
    ) {
        verify(chain)
    }

    override fun checkClientTrusted(
        chain: Array<out X509Certificate>?,
        authType: String?,
        engine: SSLEngine?,
    ) = Unit

    override fun checkServerTrusted(
        chain: Array<out X509Certificate>?,
        authType: String?,
        engine: SSLEngine?,
    ) {
        verify(chain)
    }

    override fun getAcceptedIssuers(): Array<X509Certificate> = emptyArray()

    private fun verify(chain: Array<out X509Certificate>?) {
        val cert = chain?.firstOrNull()
            ?: throw CertificateException("Пустая цепочка сертификатов сервера")
        val observedHex = SpkiPinCalculator.computeHex(cert)
        PinHolder.recordObserved(observedHex)
        val expected = PinHolder.getCurrent() ?: return
        if (!expected.equals(observedHex, ignoreCase = true)) {
            throw CertificateException(
                "SPKI pin mismatch: expected=$expected observed=$observedHex"
            )
        }
    }
}
