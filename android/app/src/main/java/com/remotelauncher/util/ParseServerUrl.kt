package com.remotelauncher.util

import java.net.URI
import java.net.URISyntaxException

sealed class ParsedUrl {
    data class Valid(val url: String) : ParsedUrl()
    data class Invalid(val reason: String) : ParsedUrl()
}

private const val DEFAULT_PORT = 8443

fun parseServerUrl(input: String): ParsedUrl {
    val trimmed = input.trim()
    if (trimmed.isEmpty()) return ParsedUrl.Invalid("адрес пустой")
    if (trimmed.any { it.isWhitespace() }) return ParsedUrl.Invalid("пробелы в адресе недопустимы")

    val withScheme = when {
        trimmed.startsWith("http://", ignoreCase = true) ->
            return ParsedUrl.Invalid("только https")
        trimmed.startsWith("https://", ignoreCase = true) -> trimmed
        else -> "https://$trimmed"
    }

    val uri = try {
        URI(withScheme)
    } catch (_: URISyntaxException) {
        return ParsedUrl.Invalid("некорректный адрес")
    }

    val host = uri.host
    if (host.isNullOrEmpty()) return ParsedUrl.Invalid("не указан хост")
    if (!isHostAllowed(host)) return ParsedUrl.Invalid("некорректный хост")

    val path = uri.rawPath.orEmpty()
    if (path.isNotEmpty() && path != "/") return ParsedUrl.Invalid("путь недопустим")
    if (!uri.rawQuery.isNullOrEmpty()) return ParsedUrl.Invalid("параметры недопустимы")
    if (!uri.rawFragment.isNullOrEmpty()) return ParsedUrl.Invalid("фрагмент недопустим")

    val port = if (uri.port == -1) DEFAULT_PORT else uri.port
    return ParsedUrl.Valid("https://$host:$port")
}

private fun isHostAllowed(host: String): Boolean {
    val stripped = host.trimStart('[').trimEnd(']')
    if (stripped.isEmpty()) return false
    return stripped.all { ch ->
        ch.isLetterOrDigit() || ch == '.' || ch == '-' || ch == ':'
    }
}
