use serde_json::Value;
use tauri_plugin_shell::process::CommandEvent;
use tauri_plugin_shell::ShellExt;

/// Runs the bundled `tuneup` sidecar with the given args and returns its
/// stdout, parsed as JSON. Any non-empty stderr is folded into the error
/// message so failures are easy to diagnose from the frontend.
pub async fn run_sidecar(app: &tauri::AppHandle, args: Vec<String>) -> Result<Value, String> {
    let sidecar = app
        .shell()
        .sidecar("tuneup")
        .map_err(|e| format!("failed to prepare tuneup: {e}"))?;

    let (mut rx, _child) = sidecar
        .args(args)
        .spawn()
        .map_err(|e| format!("failed to start tuneup: {e}"))?;

    let mut stdout = String::new();
    let mut stderr = String::new();

    while let Some(event) = rx.recv().await {
        match event {
            CommandEvent::Stdout(bytes) => {
                stdout.push_str(&String::from_utf8_lossy(&bytes));
            }
            CommandEvent::Stderr(bytes) => {
                stderr.push_str(&String::from_utf8_lossy(&bytes));
            }
            CommandEvent::Error(err) => {
                return Err(format!("tuneup process error: {err}"));
            }
            CommandEvent::Terminated(payload) => {
                if payload.code != Some(0) && !stderr.is_empty() {
                    return Err(format!("tuneup exited with an error: {stderr}"));
                }
            }
            _ => {}
        }
    }

    if stdout.trim().is_empty() {
        return Err(if stderr.is_empty() {
            "tuneup produced no output".to_string()
        } else {
            format!("tuneup failed: {stderr}")
        });
    }

    serde_json::from_str(&stdout).map_err(|e| format!("couldn't parse tuneup output: {e}"))
}

/// Runs every check, read-only.
pub async fn run_all(app: &tauri::AppHandle) -> Result<Value, String> {
    run_sidecar(app, vec!["--json".to_string()]).await
}

/// Runs a single named check, read-only (no fix applied).
pub async fn run_one(app: &tauri::AppHandle, check_name: &str) -> Result<Value, String> {
    run_sidecar(
        app,
        vec![
            "--json".to_string(),
            "--only".to_string(),
            check_name.to_string(),
        ],
    )
    .await
}

/// Re-runs a single named check with its fix applied (if it has one).
pub async fn fix_one(app: &tauri::AppHandle, check_name: &str) -> Result<Value, String> {
    run_sidecar(
        app,
        vec![
            "--json".to_string(),
            "--only".to_string(),
            check_name.to_string(),
            "--yes".to_string(),
        ],
    )
    .await
}

/// Counts how many results in a run_all()-shaped JSON value have
/// status WARNING or PROBLEM. Used by the background scheduler to
/// decide whether a notification is worth showing.
pub fn count_problems(result: &Value) -> u64 {
    result
        .get("problems_found")
        .and_then(Value::as_u64)
        .unwrap_or(0)
}
