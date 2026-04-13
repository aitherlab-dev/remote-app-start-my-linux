package com.remotelauncher.net

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockRequestHandleScope
import io.ktor.client.engine.mock.respond
import io.ktor.client.engine.mock.respondError
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.HttpRequestData
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.io.IOException

private const val BASE_URL = "https://example.test:8443"
private const val TOKEN = "test-token-xyz"

private fun MockRequestHandleScope.jsonResponse(
    body: String,
    status: HttpStatusCode = HttpStatusCode.OK
): HttpResponseData = respond(
    content = body,
    status = status,
    headers = headersOf(HttpHeaders.ContentType, "application/json")
)

private fun buildApi(
    handler: suspend MockRequestHandleScope.(HttpRequestData) -> HttpResponseData
): KtorRemoteLauncherApi {
    val engine = MockEngine { request -> handler(request) }
    val client = HttpClient(engine) {
        expectSuccess = true
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true })
        }
    }
    return KtorRemoteLauncherApi(client, BASE_URL)
}

class KtorRemoteLauncherApiTest {

    private val statusFixture = """
        {
            "version": "c99908f",
            "started_at": "2026-04-13T19:48:21.435Z",
            "uptime_sec": 6,
            "apps_count": 74,
            "cert_fingerprint": "0B:CF:14:DD:7E:13:77:5F:C5:4C:71:D6:67:AD:A6:FF:4B:40:C0:7E:03:D7:A9:D8:3E:06:A9:14:60:43:FE:D5"
        }
    """.trimIndent()

    private val appsFixture = """
        [
            {
                "id": "xfce4-about",
                "name": "About Xfce"
            },
            {
                "id": "chromium",
                "name": "Chromium",
                "comment": "Browse the web",
                "icon": "chromium",
                "categories": ["Network", "WebBrowser"],
                "running": true
            }
        ]
    """.trimIndent()

    @Test
    fun status_happyPath() = runTest {
        val api = buildApi { request ->
            assertEquals(HttpMethod.Get, request.method)
            assertEquals("$BASE_URL/api/status", request.url.toString())
            jsonResponse(statusFixture)
        }

        val result = api.status()

        assertTrue("expected Success, got $result", result is ApiResult.Success)
        val value = (result as ApiResult.Success).value
        assertEquals("c99908f", value.version)
        assertEquals("2026-04-13T19:48:21.435Z", value.startedAt)
        assertEquals(6L, value.uptimeSec)
        assertEquals(74, value.appsCount)
        assertEquals(
            "0B:CF:14:DD:7E:13:77:5F:C5:4C:71:D6:67:AD:A6:FF:4B:40:C0:7E:03:D7:A9:D8:3E:06:A9:14:60:43:FE:D5",
            value.certFingerprint
        )
    }

    @Test
    fun apps_happyPath() = runTest {
        val api = buildApi { request ->
            assertEquals(HttpMethod.Get, request.method)
            assertEquals("$BASE_URL/api/apps", request.url.toString())
            jsonResponse(appsFixture)
        }

        val result = api.apps(TOKEN)

        assertTrue("expected Success, got $result", result is ApiResult.Success)
        val apps = (result as ApiResult.Success).value
        assertEquals(2, apps.size)

        val about = apps[0]
        assertEquals("xfce4-about", about.id)
        assertEquals("About Xfce", about.name)
        assertNull(about.comment)
        assertNull(about.icon)
        assertTrue(about.categories.isEmpty())
        assertEquals(false, about.running)

        val chromium = apps[1]
        assertEquals("chromium", chromium.id)
        assertEquals("Chromium", chromium.name)
        assertEquals("Browse the web", chromium.comment)
        assertEquals("chromium", chromium.icon)
        assertEquals(listOf("Network", "WebBrowser"), chromium.categories)
        assertEquals(true, chromium.running)
    }

    @Test
    fun apps_sendsBearerHeader() = runTest {
        val api = buildApi { request ->
            assertEquals("Bearer $TOKEN", request.headers[HttpHeaders.Authorization])
            jsonResponse("[]")
        }

        val result = api.apps(TOKEN)

        assertTrue(result is ApiResult.Success)
        assertTrue((result as ApiResult.Success).value.isEmpty())
    }

    @Test
    fun apps_unauthorized() = runTest {
        val api = buildApi {
            respondError(HttpStatusCode.Unauthorized)
        }

        val result = api.apps(TOKEN)

        assertTrue("expected HttpError, got $result", result is ApiResult.HttpError)
        assertEquals(401, (result as ApiResult.HttpError).code)
    }

    @Test
    fun pair_happyPath() = runTest {
        val api = buildApi { request ->
            assertEquals(HttpMethod.Post, request.method)
            assertEquals("$BASE_URL/api/pair", request.url.toString())
            val bodyText = (request.body as io.ktor.http.content.TextContent).text
            assertTrue(
                "body must contain snake_case device_label: $bodyText",
                bodyText.contains("\"device_label\"")
            )
            assertTrue(bodyText.contains("\"pin\":\"123456\""))
            assertTrue(bodyText.contains("\"device_label\":\"test-device\""))
            jsonResponse("""{"token":"abc123"}""")
        }

        val result = api.pair(pin = "123456", deviceLabel = "test-device")

        assertTrue("expected Success, got $result", result is ApiResult.Success)
        assertEquals("abc123", (result as ApiResult.Success).value.token)
    }

    @Test
    fun pair_wrongPin() = runTest {
        val api = buildApi {
            respondError(HttpStatusCode.Unauthorized)
        }

        val result = api.pair(pin = "000000", deviceLabel = "test")

        assertTrue("expected HttpError, got $result", result is ApiResult.HttpError)
        assertEquals(401, (result as ApiResult.HttpError).code)
    }

    @Test
    fun pair_rateLimited() = runTest {
        val api = buildApi {
            respondError(HttpStatusCode.TooManyRequests)
        }

        val result = api.pair(pin = "111111", deviceLabel = "test")

        assertTrue("expected HttpError, got $result", result is ApiResult.HttpError)
        assertEquals(429, (result as ApiResult.HttpError).code)
    }

    @Test
    fun launch_happyPath() = runTest {
        val api = buildApi { request ->
            assertEquals(HttpMethod.Post, request.method)
            assertEquals("$BASE_URL/api/apps/xfce4-about/launch", request.url.toString())
            assertEquals("Bearer $TOKEN", request.headers[HttpHeaders.Authorization])
            jsonResponse("""{"status":"launched","pid":12345}""")
        }

        val result = api.launch(appId = "xfce4-about", token = TOKEN)

        assertTrue("expected Success, got $result", result is ApiResult.Success)
        val payload = (result as ApiResult.Success).value
        assertEquals("launched", payload.status)
        assertEquals(12345, payload.pid)
    }

    @Test
    fun launch_networkError() = runTest {
        val api = buildApi {
            throw IOException("boom-marker-42")
        }

        val result = api.launch(appId = "xfce4-about", token = TOKEN)

        if (result !is ApiResult.NetworkError) {
            fail("expected NetworkError, got $result")
            return@runTest
        }
        val cause = result.cause
        assertNotNull(cause)
        val hasMarker = generateSequence<Throwable>(cause) { it.cause }
            .any { it.message?.contains("boom-marker-42") == true }
        assertTrue("expected original IOException in cause chain: $cause", hasMarker)
    }
}
