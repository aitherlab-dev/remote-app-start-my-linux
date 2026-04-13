package com.remotelauncher.data

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class TokenStoreTest {

    private lateinit var store: EncryptedTokenStore

    private val urlA = "https://a.example:8443"
    private val urlB = "https://b.example:8443"

    @Before
    fun setUp() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        context.deleteSharedPreferences("tokens")
        store = EncryptedTokenStore(context)
        store.clearToken(urlA)
        store.clearToken(urlB)
    }

    @After
    fun tearDown() {
        store.clearToken(urlA)
        store.clearToken(urlB)
    }

    @Test
    fun writeAndRead_sameUrl() {
        store.setToken(urlA, "token-a")
        assertEquals("token-a", store.getToken(urlA))
    }

    @Test
    fun differentUrls_haveDifferentKeys() {
        store.setToken(urlA, "token-a")
        store.setToken(urlB, "token-b")
        assertEquals("token-a", store.getToken(urlA))
        assertEquals("token-b", store.getToken(urlB))
        assertNotEquals(store.getToken(urlA), store.getToken(urlB))
    }

    @Test
    fun clear_removesOnly_thatUrl() {
        store.setToken(urlA, "token-a")
        store.setToken(urlB, "token-b")
        store.clearToken(urlA)
        assertNull(store.getToken(urlA))
        assertEquals("token-b", store.getToken(urlB))
    }

    @Test
    fun hasToken_reflectsState() {
        assertFalse(store.hasToken(urlA))
        store.setToken(urlA, "token-a")
        assertTrue(store.hasToken(urlA))
        store.clearToken(urlA)
        assertFalse(store.hasToken(urlA))
    }

    @Test
    fun caseInsensitiveUrl_matchesKey() {
        store.setToken("https://Host.Example:8443", "token-case")
        assertEquals("token-case", store.getToken("https://host.example:8443"))
    }
}
