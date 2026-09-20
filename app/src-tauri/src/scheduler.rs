use crate::sidecar;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use tauri::AppHandle;
use tauri_plugin_notification::NotificationExt;

/// Shared, live-updatable interval (in minutes) for the background
/// scanner. 0 means disabled. Wrapped in an Arc so settings updates
/// from the frontend can change it without restarting the app.
#[derive(Clone)]
pub struct ScanInterval(pub Arc<AtomicU64>);

impl ScanInterval {
    pub fn new(minutes: u64) -> Self {
        Self(Arc::new(AtomicU64::new(minutes)))
    }

    pub fn set(&self, minutes: u64) {
        self.0.store(minutes, Ordering::Relaxed);
    }

    fn get(&self) -> u64 {
        self.0.load(Ordering::Relaxed)
    }
}

/// Spawns the background scan loop. Sleeps in short increments so a
/// live interval change (or disabling) takes effect promptly rather
/// than waiting out a long-since-stale sleep.
pub fn spawn(app: AppHandle, interval: ScanInterval) {
    tauri::async_runtime::spawn(async move {
        const POLL_SECS: u64 = 30;
        let mut elapsed_secs: u64 = 0;

        loop {
            tokio::time::sleep(std::time::Duration::from_secs(POLL_SECS)).await;
            elapsed_secs += POLL_SECS;

            let minutes = interval.get();
            if minutes == 0 {
                elapsed_secs = 0;
                continue;
            }

            if elapsed_secs < minutes * 60 {
                continue;
            }
            elapsed_secs = 0;

            match sidecar::run_all(&app).await {
                Ok(result) => {
                    let problems = sidecar::count_problems(&result);
                    if problems > 0 {
                        let body = if problems == 1 {
                            "1 check flagged something. Open tuneup for details.".to_string()
                        } else {
                            format!("{problems} checks flagged something. Open tuneup for details.")
                        };
                        let _ = app
                            .notification()
                            .builder()
                            .title("tuneup")
                            .body(body)
                            .show();
                    }
                }
                Err(_) => {
                    // A failed background scan isn't worth interrupting the
                    // user with a notification; it'll just try again next
                    // interval.
                }
            }
        }
    });
}
