package com.remotelauncher.net

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ServerStatus(
    val version: String,
    @SerialName("started_at") val startedAt: String,
    @SerialName("uptime_sec") val uptimeSec: Long,
    @SerialName("apps_count") val appsCount: Int,
    @SerialName("cert_fingerprint") val certFingerprint: String
)

@Serializable
data class AppInfo(
    val id: String,
    val name: String,
    val comment: String? = null,
    val icon: String? = null,
    val categories: List<String> = emptyList(),
    val running: Boolean = false
)

@Serializable
data class PairRequest(
    val pin: String,
    @SerialName("device_label") val deviceLabel: String
)

@Serializable
data class PairResponse(val token: String)

@Serializable
data class LaunchResponse(val status: String, val pid: Int)
