package com.remotelauncher.ui.theme

import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.TextFieldColors
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.sp

@Composable
fun fieldTextColor(): Color = MaterialTheme.colorScheme.onBackground

@Composable
fun fieldMutedColor(): Color = MaterialTheme.colorScheme.onSurfaceVariant

@Composable
fun fieldTextStyle(
    size: TextUnit = 18.sp,
    weight: FontWeight = FontWeight.Medium,
    letterSpacing: TextUnit = 0.sp,
): TextStyle = TextStyle(
    color = fieldTextColor(),
    fontSize = size,
    fontWeight = weight,
    letterSpacing = letterSpacing,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun highContrastFieldColors(): TextFieldColors {
    val scheme = MaterialTheme.colorScheme
    val text = scheme.onBackground
    val muted = scheme.onSurfaceVariant
    return OutlinedTextFieldDefaults.colors(
        focusedTextColor = text,
        unfocusedTextColor = text,
        disabledTextColor = text.copy(alpha = 0.6f),
        errorTextColor = text,
        cursorColor = scheme.primary,
        errorCursorColor = scheme.error,
        focusedBorderColor = scheme.primary,
        unfocusedBorderColor = muted,
        disabledBorderColor = muted.copy(alpha = 0.4f),
        errorBorderColor = scheme.error,
        focusedLabelColor = scheme.primary,
        unfocusedLabelColor = muted,
        disabledLabelColor = muted.copy(alpha = 0.6f),
        errorLabelColor = scheme.error,
        focusedPlaceholderColor = muted,
        unfocusedPlaceholderColor = muted,
        disabledPlaceholderColor = muted.copy(alpha = 0.6f),
        focusedContainerColor = Color.Transparent,
        unfocusedContainerColor = Color.Transparent,
        disabledContainerColor = Color.Transparent,
    )
}
