package com.remotelauncher.ui.apps

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil3.compose.AsyncImage
import coil3.network.NetworkHeaders
import coil3.network.httpHeaders
import coil3.request.ImageRequest
import coil3.request.crossfade
import coil3.size.Size
import com.remotelauncher.R
import com.remotelauncher.net.AppInfo
import kotlinx.coroutines.flow.collectLatest

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AppsScreen(
    viewModel: AppsViewModel,
    serverUrl: String,
    authToken: String,
    onUnauthorized: () -> Unit,
    onDisconnect: () -> Unit,
    onAdmin: () -> Unit = {},
    onTerminal: () -> Unit = {},
) {
    val uiState by viewModel.state.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }
    val context = LocalContext.current

    LaunchedEffect(uiState) {
        if (uiState is AppsUiState.Unauthorized) {
            onUnauthorized()
        }
    }

    LaunchedEffect(viewModel) {
        viewModel.events.collectLatest { event ->
            val message = when (event) {
                is AppsUiEvent.Launching -> context.getString(R.string.apps_launching, event.appName)
                is AppsUiEvent.Launched -> context.getString(R.string.apps_launched, event.appName)
                is AppsUiEvent.LaunchFailed -> context.getString(R.string.apps_launch_failed, event.appName, event.reason)
            }
            snackbarHostState.showSnackbar(message)
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.apps_title)) },
                actions = {
                    IconButton(onClick = onTerminal) {
                        Icon(
                            painter = painterResource(R.drawable.ic_terminal),
                            contentDescription = stringResource(R.string.apps_terminal),
                        )
                    }
                    IconButton(onClick = onAdmin) {
                        Icon(
                            imageVector = Icons.Default.Settings,
                            contentDescription = stringResource(R.string.apps_admin),
                        )
                    }
                    IconButton(onClick = { viewModel.refresh() }) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = stringResource(R.string.apps_refresh),
                        )
                    }
                    IconButton(onClick = onDisconnect) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ExitToApp,
                            contentDescription = stringResource(R.string.apps_disconnect),
                        )
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        val isRefreshing = uiState is AppsUiState.Loading
        PullToRefreshBox(
            isRefreshing = isRefreshing,
            onRefresh = { viewModel.refresh() },
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            when (val s = uiState) {
                is AppsUiState.Loading -> LoadingContent()
                is AppsUiState.Loaded -> AppsGrid(
                    apps = s.apps,
                    serverUrl = serverUrl,
                    authToken = authToken,
                    onTap = { viewModel.onTap(it) },
                )
                is AppsUiState.Empty -> EmptyContent()
                is AppsUiState.Error -> ErrorContent(
                    message = s.message,
                    onRetry = { viewModel.refresh() },
                )
                is AppsUiState.Unauthorized -> LoadingContent()
            }
        }
    }
}

@Composable
private fun LoadingContent() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        CircularProgressIndicator()
    }
}

@Composable
private fun EmptyContent() {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = stringResource(R.string.apps_empty),
            style = MaterialTheme.typography.bodyLarge,
        )
    }
}

@Composable
private fun ErrorContent(
    message: String,
    onRetry: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = message,
            color = MaterialTheme.colorScheme.error,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        Button(onClick = onRetry) {
            Text(stringResource(R.string.apps_retry))
        }
    }
}

@Composable
private fun AppsGrid(
    apps: List<AppInfo>,
    serverUrl: String,
    authToken: String,
    onTap: (AppInfo) -> Unit,
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 104.dp),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 8.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(items = apps, key = { it.id }) { app ->
            AppCard(app = app, serverUrl = serverUrl, authToken = authToken, onTap = onTap)
        }
    }
}

@Composable
private fun AppCard(
    app: AppInfo,
    serverUrl: String,
    authToken: String,
    onTap: (AppInfo) -> Unit,
) {
    val context = LocalContext.current
    val placeholder = painterResource(R.drawable.ic_app_placeholder)
    val request = ImageRequest.Builder(context)
        .data("$serverUrl/api/apps/${app.id}/icon?size=128")
        .httpHeaders(
            NetworkHeaders.Builder()
                .set("Authorization", "Bearer $authToken")
                .build()
        )
        .size(Size(128, 128))
        .crossfade(true)
        .build()

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onTap(app) },
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 12.dp, horizontal = 8.dp),
            verticalArrangement = Arrangement.Top,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            AsyncImage(
                model = request,
                contentDescription = app.name,
                placeholder = placeholder,
                error = placeholder,
                modifier = Modifier
                    .size(56.dp)
                    .clip(RoundedCornerShape(12.dp)),
            )
            Spacer(Modifier.height(8.dp))
            Text(
                text = app.name,
                maxLines = 2,
                minLines = 2,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.labelMedium,
                textAlign = TextAlign.Center,
                color = MaterialTheme.colorScheme.onSurface,
            )
        }
    }
}
