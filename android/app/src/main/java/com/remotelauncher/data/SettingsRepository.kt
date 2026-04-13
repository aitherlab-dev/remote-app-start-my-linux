package com.remotelauncher.data

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

val Context.settingsDataStore: DataStore<Preferences> by preferencesDataStore(name = "settings")

class SettingsRepository(private val dataStore: DataStore<Preferences>) {

    val serverUrl: Flow<String?> = dataStore.data.map { prefs -> prefs[KEY_SERVER_URL] }

    suspend fun setServerUrl(url: String) {
        dataStore.edit { prefs -> prefs[KEY_SERVER_URL] = url }
    }

    suspend fun clearServerUrl() {
        dataStore.edit { prefs -> prefs.remove(KEY_SERVER_URL) }
    }

    companion object {
        private val KEY_SERVER_URL = stringPreferencesKey("server_url")
    }
}
