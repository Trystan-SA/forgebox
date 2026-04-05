//! Application settings persisted via tauri-plugin-store.

use serde::{Deserialize, Serialize};
use tauri::AppHandle;
use tauri_plugin_store::StoreExt;

const STORE_FILE: &str = "settings.json";

/// User-configurable application settings.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppSettings {
    /// URL of the ForgeBox gateway to connect to.
    pub gateway_url: String,
    /// Automatically reconnect on launch using stored credentials.
    pub auto_connect: bool,
    /// UI theme: "light", "dark", or "system".
    pub theme: String,
    /// Default OAuth provider: "google", "microsoft", or "oidc".
    pub default_provider: String,
}

impl Default for AppSettings {
    fn default() -> Self {
        Self {
            gateway_url: "https://localhost:8443".to_string(),
            auto_connect: true,
            theme: "system".to_string(),
            default_provider: "google".to_string(),
        }
    }
}

/// Load settings from the store, falling back to defaults.
pub fn load_settings(app: &AppHandle) -> AppSettings {
    let store = match app.store(STORE_FILE) {
        Ok(s) => s,
        Err(_) => return AppSettings::default(),
    };

    let get_str = |key: &str, default: &str| -> String {
        store
            .get(key)
            .and_then(|v| v.as_str().map(String::from))
            .unwrap_or_else(|| default.to_string())
    };

    let get_bool = |key: &str, default: bool| -> bool {
        store
            .get(key)
            .and_then(|v| v.as_bool())
            .unwrap_or(default)
    };

    let defaults = AppSettings::default();
    AppSettings {
        gateway_url: get_str("gateway_url", &defaults.gateway_url),
        auto_connect: get_bool("auto_connect", defaults.auto_connect),
        theme: get_str("theme", &defaults.theme),
        default_provider: get_str("default_provider", &defaults.default_provider),
    }
}

/// Persist settings to the store.
pub fn save_settings(app: &AppHandle, settings: &AppSettings) {
    let Ok(store) = app.store(STORE_FILE) else {
        return;
    };

    store.set("gateway_url", serde_json::json!(&settings.gateway_url));
    store.set("auto_connect", serde_json::json!(settings.auto_connect));
    store.set("theme", serde_json::json!(&settings.theme));
    store.set("default_provider", serde_json::json!(&settings.default_provider));
}
