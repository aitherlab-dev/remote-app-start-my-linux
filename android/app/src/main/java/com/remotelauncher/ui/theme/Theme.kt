package com.remotelauncher.ui.theme

import android.app.Activity
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

private val LightColors = lightColorScheme(
    primary = AccentLight,
    onPrimary = Color.White,
    primaryContainer = Color(0xFFDDE6FB),
    onPrimaryContainer = TextLight,
    secondary = TextMutedLight,
    onSecondary = Color.White,
    background = BackgroundLight,
    onBackground = TextLight,
    surface = BackgroundLight,
    onSurface = TextLight,
    surfaceVariant = SurfaceVariantLight,
    onSurfaceVariant = TextMutedLight,
    outline = OutlineLight,
    outlineVariant = OutlineLight,
    error = ErrorLight,
    onError = Color.White,
)

private val DarkColors = darkColorScheme(
    primary = AccentDark,
    onPrimary = Color(0xFF00174B),
    primaryContainer = Color(0xFF1E2F5A),
    onPrimaryContainer = TextDark,
    secondary = TextMutedDark,
    onSecondary = Color(0xFF0E0F12),
    background = BackgroundDark,
    onBackground = TextDark,
    surface = BackgroundDark,
    onSurface = TextDark,
    surfaceVariant = SurfaceVariantDark,
    onSurfaceVariant = TextMutedDark,
    outline = OutlineDark,
    outlineVariant = OutlineDark,
    error = ErrorDark,
    onError = Color(0xFF1A0000),
)

@Composable
fun RemoteLauncherTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    val colorScheme = if (darkTheme) DarkColors else LightColors
    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            window.statusBarColor = colorScheme.background.toArgb()
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = !darkTheme
        }
    }
    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content,
    )
}
