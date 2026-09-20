#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod scheduler;
mod settings;
mod sidecar;
mod tray;

use scheduler::ScanInterval;
use settings::AppSettings;
use tauri::Manager;
use tauri_plugin_autostart::ManagerExt as AutostartManagerExt;

#[tauri::command]
async fn run_checks(app: tauri::AppHandle) -> Result<serde_json::Value, String> {
    sidecar::run_all(&app).await
}

#[tauri::command]
async fn run_check(app: tauri::AppHandle, check_name: String) -> Result<serde_json::Value, String> {
    sidecar::run_one(&app, &check_name).await
}

#[tauri::command]
async fn apply_fix(app: tauri::AppHandle, check_name: String) -> Result<serde_json::Value, String> {
    sidecar::fix_one(&app, &check_name).await
}

#[tauri::command]
fn get_settings(app: tauri::AppHandle) -> AppSettings {
    settings::load(&app)
}

#[tauri::command]
fn save_settings(app: tauri::AppHandle, new_settings: AppSettings) -> Result<(), String> {
    settings::save(&app, &new_settings)?;

    // Apply side effects of whatever changed.
    if let Some(interval_state) = app.try_state::<ScanInterval>() {
        interval_state.set(new_settings.scan_interval_minutes);
    }

    if new_settings.launch_on_login {
        let _ = app.autolaunch().enable();
    } else {
        let _ = app.autolaunch().disable();
    }

    if new_settings.show_in_tray {
        // Re-creating an existing tray id is a no-op error we can
        // safely ignore; this keeps the toggle idempotent.
        let _ = tray::create_tray(&app);
    } else {
        tray::remove_tray(&app);
    }

    Ok(())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            None,
        ))
        .invoke_handler(tauri::generate_handler![
            run_checks,
            run_check,
            apply_fix,
            get_settings,
            save_settings
        ])
        .setup(|app| {
            let handle = app.handle().clone();
            let initial = settings::load(&handle);

            if initial.show_in_tray {
                if let Err(e) = tray::create_tray(&handle) {
                    eprintln!("failed to create tray icon: {e}");
                }
            }

            if initial.launch_on_login {
                let _ = handle.autolaunch().enable();
            }

            let interval = ScanInterval::new(initial.scan_interval_minutes);
            app.manage(interval.clone());
            scheduler::spawn(handle.clone(), interval);

            // Closing the window hides it instead of quitting, so the
            // app keeps running in the tray (only meaningful when the
            // tray is actually enabled; otherwise closing just quits
            // via the default OS behavior for a hidden, unreachable
            // window, so we still exit in that case).
            if let Some(window) = app.get_webview_window("main") {
                let handle_for_close = handle.clone();
                window.on_window_event(move |event| {
                    if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                        let settings = settings::load(&handle_for_close);
                        if settings.show_in_tray {
                            if let Some(w) = handle_for_close.get_webview_window("main") {
                                let _ = w.hide();
                            }
                            api.prevent_close();
                        }
                    }
                });
            }

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tuneup-app");
}
