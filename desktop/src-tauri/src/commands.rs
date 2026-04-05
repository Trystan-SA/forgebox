//! Tauri command handlers exposed to the frontend via `invoke()`.

use crate::auth;
use crate::settings::{self, AppSettings};
use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use tauri::{AppHandle, State};
use thiserror::Error;

// ---------------------------------------------------------------------------
// Error type
// ---------------------------------------------------------------------------

#[derive(Debug, Error)]
pub enum CommandError {
    #[error("{0}")]
    Auth(#[from] anyhow::Error),
    #[error("lock poisoned")]
    Lock,
}

// Tauri commands must return a type that implements `Serialize`.
impl Serialize for CommandError {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(&self.to_string())
    }
}

// ---------------------------------------------------------------------------
// Shared state
// ---------------------------------------------------------------------------

/// Managed state holding the current authentication info (if any).
pub struct AppState {
    pub auth: Mutex<Option<AuthInfo>>,
}

/// Minimal authentication info surfaced to the frontend.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthInfo {
    pub email: String,
    pub name: String,
    pub provider: String,
    pub gateway_url: String,
}

impl From<auth::AuthState> for AuthInfo {
    fn from(s: auth::AuthState) -> Self {
        Self {
            email: s.user_email,
            name: s.user_name,
            provider: s.provider,
            gateway_url: s.gateway_url,
        }
    }
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

/// Connect to a ForgeBox gateway, restoring a previously-stored session if one
/// exists in the OS keychain.
#[tauri::command]
pub async fn connect(
    gateway_url: String,
    state: State<'_, AppState>,
) -> Result<AuthInfo, CommandError> {
    // Try to load an existing session from the keychain.
    if let Some(stored) = auth::load_stored_auth() {
        if stored.gateway_url == gateway_url {
            let info = AuthInfo::from(stored);
            *state.auth.lock().map_err(|_| CommandError::Lock)? = Some(info.clone());
            return Ok(info);
        }
    }

    Err(anyhow::anyhow!(
        "no stored session for {gateway_url} — use start_oauth to authenticate"
    )
    .into())
}

/// Disconnect from the current ForgeBox gateway and clear stored credentials.
#[tauri::command]
pub async fn disconnect(state: State<'_, AppState>) -> Result<(), CommandError> {
    auth::clear_auth();
    *state.auth.lock().map_err(|_| CommandError::Lock)? = None;
    Ok(())
}

/// Return the current authentication state, or `null` if not authenticated.
#[tauri::command]
pub async fn get_auth_state(
    state: State<'_, AppState>,
) -> Result<Option<AuthInfo>, CommandError> {
    let guard = state.auth.lock().map_err(|_| CommandError::Lock)?;
    Ok(guard.clone())
}

/// Initiate the OAuth / OIDC login flow for the given provider.
///
/// Opens the system browser and waits for the callback. On success the tokens
/// are stored in the OS keychain and the user info is returned.
#[tauri::command]
pub async fn start_oauth(
    provider: String,
    gateway_url: String,
    state: State<'_, AppState>,
) -> Result<AuthInfo, CommandError> {
    let auth_state = auth::start_oauth_flow(&provider, &gateway_url).await?;
    let info = AuthInfo::from(auth_state);
    *state.auth.lock().map_err(|_| CommandError::Lock)? = Some(info.clone());
    Ok(info)
}

/// Persist application settings.
#[tauri::command]
pub async fn save_settings(
    app: AppHandle,
    new_settings: AppSettings,
) -> Result<(), CommandError> {
    settings::save_settings(&app, &new_settings);
    Ok(())
}

/// Load application settings.
#[tauri::command]
pub async fn get_settings(app: AppHandle) -> Result<AppSettings, CommandError> {
    Ok(settings::load_settings(&app))
}
