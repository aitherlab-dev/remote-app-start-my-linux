package com.remotelauncher.ui.connect

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.remotelauncher.R

@Composable
fun ConnectScreen(
    viewModel: ConnectViewModel,
    onConnected: (serverUrl: String) -> Unit = {},
) {
    val uiState by viewModel.state.collectAsStateWithLifecycle()
    val savedUrl by viewModel.savedUrl.collectAsStateWithLifecycle()

    var input by remember { mutableStateOf("") }
    var prefilled by remember { mutableStateOf(false) }

    LaunchedEffect(savedUrl) {
        if (!prefilled && savedUrl.isNotEmpty()) {
            input = savedUrl
            prefilled = true
        }
    }

    LaunchedEffect(uiState) {
        val s = uiState
        if (s is ConnectUiState.Connected) {
            onConnected(s.serverUrl)
        }
    }

    val isConnecting = uiState is ConnectUiState.Connecting
    val isPinDialogOpen = uiState is ConnectUiState.PinConfirmRequired
    val inputDisabled = isConnecting || isPinDialogOpen

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.connect_title),
            style = MaterialTheme.typography.headlineSmall,
        )
        Spacer(Modifier.height(24.dp))
        OutlinedTextField(
            value = input,
            onValueChange = {
                input = it
                if (uiState is ConnectUiState.InputError ||
                    uiState is ConnectUiState.ConnectionFailed
                ) {
                    viewModel.reset()
                }
            },
            label = { Text(stringResource(R.string.connect_address_label)) },
            placeholder = { Text(stringResource(R.string.connect_address_placeholder)) },
            singleLine = true,
            enabled = !inputDisabled,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(16.dp))
        Button(
            onClick = { viewModel.connect(input) },
            enabled = !inputDisabled,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(stringResource(R.string.connect_button))
        }
        Spacer(Modifier.height(16.dp))

        when (val s = uiState) {
            is ConnectUiState.Idle -> Unit
            is ConnectUiState.Connecting -> {
                CircularProgressIndicator()
                Spacer(Modifier.height(8.dp))
                Text(stringResource(R.string.connect_connecting))
            }
            is ConnectUiState.InputError -> {
                Text(
                    text = s.message,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            is ConnectUiState.ConnectionFailed -> {
                Text(
                    text = s.message,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            is ConnectUiState.PinConfirmRequired -> Unit
            is ConnectUiState.Connected -> {
                Text(
                    text = stringResource(
                        R.string.connect_success,
                        s.status.version,
                        s.status.appsCount,
                    ),
                )
            }
        }
    }

    val pinState = uiState
    if (pinState is ConnectUiState.PinConfirmRequired) {
        AlertDialog(
            onDismissRequest = { viewModel.dismissPin() },
            title = { Text(stringResource(R.string.pin_confirm_title)) },
            text = {
                Column {
                    Text(stringResource(R.string.pin_confirm_message))
                    Spacer(Modifier.height(12.dp))
                    Text(
                        text = stringResource(R.string.pin_fingerprint_label),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = pinState.displayFingerprint,
                        style = MaterialTheme.typography.bodyMedium,
                        fontFamily = FontFamily.Monospace,
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = { viewModel.confirmPin() }) {
                    Text(stringResource(R.string.pin_confirm_trust))
                }
            },
            dismissButton = {
                TextButton(onClick = { viewModel.dismissPin() }) {
                    Text(stringResource(R.string.pin_confirm_cancel))
                }
            },
        )
    }
}
