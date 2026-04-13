package com.remotelauncher.data

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.remotelauncher.util.sha256Hex

interface TokenStore {
    fun getToken(serverUrl: String): String?
    fun setToken(serverUrl: String, token: String)
    fun clearToken(serverUrl: String)
    fun hasToken(serverUrl: String): Boolean = getToken(serverUrl) != null
}

class EncryptedTokenStore(context: Context) : TokenStore {

    private val prefs = EncryptedSharedPreferences.create(
        context.applicationContext,
        PREFS_NAME,
        MasterKey.Builder(context.applicationContext)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    override fun getToken(serverUrl: String): String? =
        prefs.getString(keyFor(serverUrl), null)

    override fun setToken(serverUrl: String, token: String) {
        prefs.edit().putString(keyFor(serverUrl), token).apply()
    }

    override fun clearToken(serverUrl: String) {
        prefs.edit().remove(keyFor(serverUrl)).apply()
    }

    private fun keyFor(serverUrl: String): String =
        "token_" + sha256Hex(serverUrl.lowercase())

    companion object {
        private const val PREFS_NAME = "tokens"
    }
}
