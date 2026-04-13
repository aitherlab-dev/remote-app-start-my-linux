package com.remotelauncher.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ParseServerUrlTest {

    @Test
    fun empty_input_is_invalid() {
        val r = parseServerUrl("")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun whitespace_only_is_invalid() {
        val r = parseServerUrl("   ")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun whitespace_inside_host_is_invalid() {
        val r = parseServerUrl("my host.com")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun bare_host_gets_scheme_and_default_port() {
        val r = parseServerUrl("localhost")
        assertEquals(ParsedUrl.Valid("https://localhost:8443"), r)
    }

    @Test
    fun host_with_port_gets_scheme_preserved() {
        val r = parseServerUrl("localhost:9000")
        assertEquals(ParsedUrl.Valid("https://localhost:9000"), r)
    }

    @Test
    fun https_with_port_is_canonical() {
        val r = parseServerUrl("https://localhost:8443")
        assertEquals(ParsedUrl.Valid("https://localhost:8443"), r)
    }

    @Test
    fun https_without_port_adds_default() {
        val r = parseServerUrl("https://example.com")
        assertEquals(ParsedUrl.Valid("https://example.com:8443"), r)
    }

    @Test
    fun http_scheme_is_invalid() {
        val r = parseServerUrl("http://localhost:8443")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun http_scheme_case_insensitive_is_invalid() {
        val r = parseServerUrl("HTTP://foo.com")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun ipv4_bare_is_valid() {
        val r = parseServerUrl("192.168.1.10")
        assertEquals(ParsedUrl.Valid("https://192.168.1.10:8443"), r)
    }

    @Test
    fun ipv4_with_port_is_valid() {
        val r = parseServerUrl("192.168.1.10:8443")
        assertEquals(ParsedUrl.Valid("https://192.168.1.10:8443"), r)
    }

    @Test
    fun domain_with_port_is_valid() {
        val r = parseServerUrl("mydomain.com:9443")
        assertEquals(ParsedUrl.Valid("https://mydomain.com:9443"), r)
    }

    @Test
    fun garbage_colons_is_invalid() {
        val r = parseServerUrl("::::")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun path_in_url_is_invalid() {
        val r = parseServerUrl("https://foo.com/path")
        assertTrue(r is ParsedUrl.Invalid)
    }

    @Test
    fun trailing_slash_is_allowed() {
        val r = parseServerUrl("https://foo.com:8443/")
        assertEquals(ParsedUrl.Valid("https://foo.com:8443"), r)
    }

    @Test
    fun leading_and_trailing_whitespace_is_trimmed() {
        val r = parseServerUrl("   localhost:8443   ")
        assertEquals(ParsedUrl.Valid("https://localhost:8443"), r)
    }
}
