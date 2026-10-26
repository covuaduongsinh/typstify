//! Typstify app: one Tauri shell for desktop and mobile.
//!
//! - Desktop (`desktop.rs`): runs typstify-server as a sidecar and shows
//!   its web UI (docs/plans/plan_phase7b_tauri_desktop.md).
//! - Mobile (`mobile.rs`): the web UI is bundled in the app and talks to a
//!   self-hosted server through a local proxy (`proxy.rs`,
//!   docs/plans/plan_phase7c_tauri_mobile.md).

use std::fs;
use std::net::TcpListener;

use tauri::{AppHandle, Manager, Url};
use tauri_plugin_opener::OpenerExt;

#[cfg(desktop)]
mod desktop;
#[cfg(mobile)]
mod mobile;
// Built on desktop too, for its tests.
#[cfg_attr(not(mobile), allow(dead_code))]
mod proxy;

/// Binds 127.0.0.1 on the port used last time, or on a new free one (then
/// remembered). The web UI's browser storage (preferences, offline edits in
/// IndexedDB, cached fonts) belongs to its origin, port included, so the
/// port must stay the same across launches.
pub(crate) fn bind_remembered_port(app: &AppHandle) -> std::io::Result<TcpListener> {
    let saved = app.path().app_config_dir().ok().map(|d| d.join("port.txt"));
    if let Some(port) = saved
        .as_ref()
        .and_then(|f| fs::read_to_string(f).ok())
        .and_then(|s| s.trim().parse::<u16>().ok())
    {
        if let Ok(l) = TcpListener::bind(("127.0.0.1", port)) {
            return Ok(l);
        }
    }
    let listener = TcpListener::bind(("127.0.0.1", 0))?;
    if let Some(file) = saved {
        if let Some(dir) = file.parent() {
            let _ = fs::create_dir_all(dir);
        }
        let _ = fs::write(file, listener.local_addr()?.port().to_string());
    }
    Ok(listener)
}

/// Whether a URL is a page of the app (the local server or proxy), as
/// opposed to a link out of a document or package page.
pub(crate) fn is_app_url(url: &Url, port: u16) -> bool {
    url.scheme() == "http" && url.host_str() == Some("127.0.0.1") && url.port() == Some(port)
}

/// Links out of the app open in the system browser, never in its window
/// (which has no address bar).
pub(crate) fn open_outside(app: &AppHandle, url: &Url) {
    if matches!(url.scheme(), "http" | "https" | "mailto") {
        let _ = app.opener().open_url(url.as_str(), None::<&str>);
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    #[cfg(desktop)]
    desktop::run();
    #[cfg(mobile)]
    mobile::run();
}
