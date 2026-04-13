package com.remotelauncher.net

interface RemoteLauncherApi {
    suspend fun status(): ApiResult<ServerStatus>
    suspend fun apps(token: String): ApiResult<List<AppInfo>>
    suspend fun pair(pin: String, deviceLabel: String): ApiResult<PairResponse>
    suspend fun launch(appId: String, token: String): ApiResult<LaunchResponse>
}
