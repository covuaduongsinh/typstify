//! The mobile app (docs/plans/plan_phase7c_tauri_mobile.md): the bundled
//! web UI, served with the self-hosted server's API by the local proxy
//! (src/proxy.rs).
//!
//! The server is chosen on a page of the app itself (app-setup.html, served
//! by Tauri, the only page allowed to call `connect`); the proxy has no HTTP
//! route for it, so other apps on the phone cannot point it at a server of
//! theirs.

use std::collections::HashSet;
use std::fs;
use std::sync::Arc;

use tauri::webview::NewWindowResponse;
use tauri::{AppHandle, Manager, State, Url, WebviewUrl, WebviewWindowBuilder};

use crate::proxy::{self, Proxy, Upstream, APP_PREFIX};
use crate::{bind_remembered_port, is_app_url, open_outside};

/// The setup page, as the WebView sees it.
#[cfg(target_os = "android")]
const SETUP_URL: &str = "http://tauri.localhost/app-setup.html";
#[cfg(not(target_os = "android"))]
const SETUP_URL: &str = "tauri://localhost/app-setup.html";

const SERVER_FILE: &str = "server.txt";

struct Mobile {
    proxy: Arc<Proxy>,
    port: u16,
}

fn proxy_url(port: u16) -> Url {
    Url::parse(&format!("http://127.0.0.1:{port}/")).expect("valid proxy URL")
}

fn is_setup_page(url: &Url) -> bool {
    matches!(url.scheme(), "tauri" | "http" | "https")
        && matches!(url.host_str(), Some("localhost") | Some("tauri.localhost"))
        && url.port().is_none()
}

fn load_server(app: &AppHandle) -> Option<Upstream> {
    let file = app.path().app_config_dir().ok()?.join(SERVER_FILE);
    Upstream::parse(&fs::read_to_string(file).ok()?).ok()
}

fn save_server(app: &AppHandle, up: &Upstream) -> Result<(), String> {
    let dir = app.path().app_config_dir().map_err(|e| e.to_string())?;
    fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    fs::write(dir.join(SERVER_FILE), up.origin()).map_err(|e| e.to_string())
}

fn navigate(app: &AppHandle, url: Url) {
    if let Some(w) = app.get_webview_window("main") {
        let _ = w.navigate(url);
    }
}

/// The server the app uses now, to prefill the setup page.
#[tauri::command]
fn current_server(state: State<'_, Mobile>) -> Option<String> {
    state.proxy.upstream().map(|u| u.origin())
}

/// Checks and saves the server, then opens the app on it.
#[tauri::command]
async fn connect(app: AppHandle, state: State<'_, Mobile>, url: String) -> Result<(), String> {
    let up = Upstream::parse(&url)?;
    state.proxy.probe(&up).await?;
    let previous = state.proxy.upstream();
    if previous.as_ref().is_some_and(|p| *p != up) {
        // The session cookie and the offline data belong to 127.0.0.1, not
        // to a server: kept, the new server would receive the old one's
        // session cookie, and old offline edits could be pushed to it.
        if let Some(w) = app.get_webview_window("main") {
            w.clear_all_browsing_data().map_err(|e| e.to_string())?;
        }
    }
    save_server(&app, &up)?;
    state.proxy.set_upstream(Some(up));
    navigate(&app, proxy_url(state.port));
    Ok(())
}

/// Every bundled file, read through Tauri's assets. Tauri falls back to
/// index.html for any unknown path, so the proxy only asks for real files.
fn bundled_assets(app: &AppHandle) -> proxy::Assets {
    let resolver = app.asset_resolver();
    let names: HashSet<String> = resolver
        .iter()
        .map(|(name, _)| name.trim_start_matches('/').to_string())
        .collect();
    let app = app.clone();
    Box::new(move |path| {
        if !names.contains(path) || path == "app-setup.html" {
            return None;
        }
        app.asset_resolver().get(path.to_string()).map(|a| a.bytes().to_vec())
    })
}

fn setup(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let listener = bind_remembered_port(app)?;
    let port = listener.local_addr()?.port();
    listener.set_nonblocking(true)?;

    let proxy = Arc::new(Proxy::new(bundled_assets(app)));
    let start = match load_server(app) {
        Some(up) => {
            proxy.set_upstream(Some(up));
            WebviewUrl::External(proxy_url(port))
        }
        None => WebviewUrl::App("app-setup.html".into()),
    };
    let served = proxy.clone();
    tauri::async_runtime::spawn(async move {
        if let Ok(l) = tokio::net::TcpListener::from_std(listener) {
            proxy::serve(l, served).await;
        }
    });
    app.manage(Mobile { proxy, port });

    let nav_app = app.clone();
    let popup_app = app.clone();
    WebviewWindowBuilder::new(app, "main", start)
        .on_navigation(move |url| {
            if is_app_url(url, port) {
                if url.path().starts_with(APP_PREFIX) {
                    // "Đổi máy chủ" in the web UI.
                    if url.path() == "/__app/server" {
                        let app = nav_app.clone();
                        tauri::async_runtime::spawn(async move {
                            navigate(&app, Url::parse(SETUP_URL).expect("valid setup URL"));
                        });
                    }
                    return false;
                }
                return true;
            }
            if is_setup_page(url) {
                return true;
            }
            open_outside(&nav_app, url);
            false
        })
        .on_new_window(move |url, _features| {
            // An exported PDF and links out: the phone's own apps show them
            // better than a second WebView without controls.
            open_outside(&popup_app, &url);
            NewWindowResponse::Deny
        })
        .build()?;
    Ok(())
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![connect, current_server])
        .setup(|app| setup(app.handle()))
        .run(tauri::generate_context!())
        .expect("error while running Typstify");
}
