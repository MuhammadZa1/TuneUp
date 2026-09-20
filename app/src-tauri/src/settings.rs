use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use tauri::Manager;

/// Persisted user settings. `scan_interval_minutes` of 0 means
/// automatic background scanning is off.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppSettings {
    pub theme: String, // "light" or "dark"
    pub show_in_tray: bool,
    pub launch_on_login: bool,
    pub scan_interval_minutes: u64,
}

impl Default for AppSettings {
    fn default() -> Self {
        Self {
            theme: "light".to_string(),
            show_in_tray: true,
            launch_on_login: false,
            scan_interval_minutes: 0,
        }
    }
}

fn settings_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("couldn't resolve app data directory: {e}"))?;
    fs::create_dir_all(&dir).map_err(|e| format!("couldn't create app data directory: {e}"))?;
    Ok(dir.join("settings.json"))
}

pub fn load(app: &tauri::AppHandle) -> AppSettings {
    let path = match settings_path(app) {
        Ok(p) => p,
        Err(_) => return AppSettings::default(),
    };
    match fs::read_to_string(&path) {
        Ok(contents) => serde_json::from_str(&contents).unwrap_or_default(),
        Err(_) => AppSettings::default(),
    }
}

pub fn save(app: &tauri::AppHandle, settings: &AppSettings) -> Result<(), String> {
    let path = settings_path(app)?;
    let json = serde_json::to_string_pretty(settings)
        .map_err(|e| format!("couldn't serialize settings: {e}"))?;
    fs::write(&path, json).map_err(|e| format!("couldn't write settings file: {e}"))
}
