// Prevents an extra console window on Windows in release builds.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod auth;
mod commands;
mod settings;

use commands::AppState;
use std::sync::Mutex;
use tauri::{
    menu::{MenuBuilder, MenuItemBuilder},
    tray::TrayIconBuilder,
    Manager,
};

fn main() {
    tauri::Builder::default()
        // -- Tauri plugins --------------------------------------------------
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        // -- Managed state --------------------------------------------------
        .manage(AppState {
            auth: Mutex::new(None),
        })
        // -- Commands -------------------------------------------------------
        .invoke_handler(tauri::generate_handler![
            commands::connect,
            commands::disconnect,
            commands::get_auth_state,
            commands::start_oauth,
            commands::save_settings,
            commands::get_settings,
        ])
        // -- Setup (tray icon, auto-connect) --------------------------------
        .setup(|app| {
            // Build tray menu items.
            let show = MenuItemBuilder::with_id("show", "Show ForgeBox").build(app)?;
            let run_task = MenuItemBuilder::with_id("run_task", "Run Task...").build(app)?;
            let settings = MenuItemBuilder::with_id("settings", "Settings").build(app)?;
            let quit = MenuItemBuilder::with_id("quit", "Quit").build(app)?;

            let menu = MenuBuilder::new(app)
                .item(&show)
                .separator()
                .item(&run_task)
                .item(&settings)
                .separator()
                .item(&quit)
                .build()?;

            TrayIconBuilder::new()
                .menu(&menu)
                .tooltip("ForgeBox")
                .on_menu_event(|app, event| match event.id().as_ref() {
                    "show" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                    "run_task" => {
                        // Emit an event so the frontend can open the task dialog.
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.emit("tray-run-task", ());
                        }
                    }
                    "settings" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.emit("tray-open-settings", ());
                        }
                    }
                    "quit" => {
                        app.exit(0);
                    }
                    _ => {}
                })
                .build(app)?;

            // Attempt auto-connect with stored credentials on startup.
            let app_handle = app.handle().clone();
            tauri::async_runtime::spawn(async move {
                let settings = settings::load_settings(&app_handle);
                if settings.auto_connect {
                    if let Some(stored) = auth::load_stored_auth() {
                        if stored.gateway_url == settings.gateway_url {
                            let info = commands::AuthInfo::from(stored);
                            if let Some(state) = app_handle.try_state::<AppState>() {
                                if let Ok(mut guard) = state.auth.lock() {
                                    *guard = Some(info);
                                }
                            }
                            // Notify the frontend that authentication is ready.
                            if let Some(window) = app_handle.get_webview_window("main") {
                                let _ = window.emit("auth-restored", ());
                            }
                        }
                    }
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("failed to run ForgeBox desktop app");
}
