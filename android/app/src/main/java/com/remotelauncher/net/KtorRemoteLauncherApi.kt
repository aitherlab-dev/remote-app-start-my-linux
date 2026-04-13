package com.remotelauncher.net

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.ClientRequestException
import io.ktor.client.plugins.ResponseException
import io.ktor.client.plugins.ServerResponseException
import io.ktor.client.request.bearerAuth
import io.ktor.client.request.get
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.http.ContentType
import io.ktor.http.contentType

class KtorRemoteLauncherApi(
    private val client: HttpClient,
    private val baseUrl: String
) : RemoteLauncherApi {

    override suspend fun status(): ApiResult<ServerStatus> = safeCall {
        client.get("$baseUrl/api/status").body()
    }

    override suspend fun apps(token: String): ApiResult<List<AppInfo>> = safeCall {
        client.get("$baseUrl/api/apps") {
            bearerAuth(token)
        }.body()
    }

    override suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse> = safeCall {
        client.post("$baseUrl/api/pair") {
            contentType(ContentType.Application.Json)
            setBody(PairRequest(pin = pin, deviceLabel = deviceLabel))
        }.body()
    }

    override suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse> = safeCall {
        client.post("$baseUrl/api/apps/$appId/launch") {
            bearerAuth(token)
        }.body()
    }

    private suspend inline fun <T> safeCall(block: () -> T): ApiResult<T> = try {
        ApiResult.Success(block())
    } catch (e: ClientRequestException) {
        ApiResult.HttpError(e.response.status.value, e.message)
    } catch (e: ServerResponseException) {
        ApiResult.HttpError(e.response.status.value, e.message)
    } catch (e: ResponseException) {
        ApiResult.HttpError(e.response.status.value, e.message)
    } catch (e: Throwable) {
        ApiResult.NetworkError(e)
    }
}
