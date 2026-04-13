package com.remotelauncher.data

import com.remotelauncher.net.ApiResult
import com.remotelauncher.net.AppInfo
import com.remotelauncher.net.RemoteLauncherApi

class AppsRepository(
    private val apiFactory: (String) -> RemoteLauncherApi,
    private val tokenStore: TokenStore,
    private val serverUrl: String,
) {
    suspend fun listApps(): ApiResult<List<AppInfo>> {
        val token = tokenStore.getToken(serverUrl)
            ?: return ApiResult.HttpError(401, "no token")
        return apiFactory(serverUrl).apps(token)
    }
}
