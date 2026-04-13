package com.remotelauncher.util

import java.security.MessageDigest

fun sha256Hex(input: String): String {
    val digest = MessageDigest.getInstance("SHA-256").digest(input.toByteArray(Charsets.UTF_8))
    val sb = StringBuilder(digest.size * 2)
    for (b in digest) {
        val v = b.toInt() and 0xff
        sb.append(HEX[v ushr 4])
        sb.append(HEX[v and 0x0f])
    }
    return sb.toString()
}

private val HEX = "0123456789abcdef".toCharArray()
