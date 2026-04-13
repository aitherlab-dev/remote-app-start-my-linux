package com.remotelauncher.net

import java.security.MessageDigest
import java.security.cert.X509Certificate

object SpkiPinCalculator {
    fun computeHex(cert: X509Certificate): String {
        val spkiDer = cert.publicKey.encoded
        val digest = MessageDigest.getInstance("SHA-256").digest(spkiDer)
        return digest.joinToString("") { "%02X".format(it) }
    }

    fun toFingerprint(hex: String): String =
        hex.chunked(2).joinToString(":")
}
