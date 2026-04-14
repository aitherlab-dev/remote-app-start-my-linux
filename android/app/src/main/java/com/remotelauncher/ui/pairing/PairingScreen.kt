package com.remotelauncher.ui.pairing

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.remotelauncher.R
import com.remotelauncher.ui.theme.fieldTextStyle
import com.remotelauncher.ui.theme.highContrastFieldColors

@Composable
fun PairingScreen(
    viewModel: PairingViewModel,
    onPaired: () -> Unit,
) {
    val uiState by viewModel.state.collectAsStateWithLifecycle()
    var pin by remember { mutableStateOf("") }

    val isSending = uiState is PairingUiState.Sending
    val canSubmit = !isSending && pin.length == PairingViewModel.PIN_LENGTH && pin.all { it.isDigit() }

    LaunchedEffect(uiState) {
        if (uiState is PairingUiState.Paired) {
            onPaired()
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.pairing_title),
            style = MaterialTheme.typography.headlineSmall,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.pairing_hint),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(24.dp))
        OutlinedTextField(
            value = pin,
            onValueChange = { raw ->
                val filtered = raw.filter { it.isDigit() }.take(PairingViewModel.PIN_LENGTH)
                pin = filtered
                if (uiState is PairingUiState.Error) viewModel.reset()
            },
            label = { Text(stringResource(R.string.pairing_pin_label)) },
            singleLine = true,
            enabled = !isSending,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
            textStyle = fieldTextStyle(
                size = 22.sp,
                weight = FontWeight.SemiBold,
                letterSpacing = 4.sp,
            ),
            colors = highContrastFieldColors(),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(16.dp))
        Button(
            onClick = { viewModel.submit(pin) },
            enabled = canSubmit,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(stringResource(R.string.pairing_submit))
        }
        Spacer(Modifier.height(16.dp))

        when (val s = uiState) {
            is PairingUiState.EnterPin, PairingUiState.Paired -> Unit
            is PairingUiState.Sending -> {
                CircularProgressIndicator()
                Spacer(Modifier.height(8.dp))
                Text(stringResource(R.string.pairing_sending))
            }
            is PairingUiState.Error -> {
                Text(
                    text = s.message,
                    color = MaterialTheme.colorScheme.error,
                )
            }
        }
    }
}
