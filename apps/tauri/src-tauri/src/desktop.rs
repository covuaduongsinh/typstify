// Typstify desktop: a Tauri shell around typstify-server.
//
// The window shows the same React UI as the web version, served by a
// typstify-server sidecar on 127.0.0.1. A random launch token, passed to
// the server through the environment, signs the window in once
// (server/launch.go); in this mode the server always requires
// authentication and checks Host/Origin, so other pages on this machine
// cannot use the local port. See docs/plans/plan_phase7b_tauri_desktop.md.
use std::fs::{self, File};
use std::io::{BufRead, BufReader, Write};
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::sync::mpsc::{self, Receiver, RecvTimeoutError};
use std::sync::{Mutex, MutexGuard};
use std::thread;
use std::time::{Duration, Instant};

use tauri::webview::NewWindowResponse;
use tauri::{AppHandle, Manager, RunEvent, WebviewUrl, WebviewWindowBuilder};

use crate::{bind_remembered_port, is_app_url, open_outside};

/// How long the server may take to start listening.
const START_TIMEOUT: Duration = Duration::from_secs(60);

/// The line the server prints on stdout once it holds the port
/// (cmd/typstify-server/main.go).
const READY_PREFIX: &str = "TYPSTIFY_READY ";

/// The running server; dropped (and with it, on Windows, the job that kills
/// the whole process tree) when the app exits.
struct Sidecar(Mutex<Option<Server>>);

struct Server {
    child: Child,
    #[cfg(windows)]
    _job: job::KillOnClose,
}

impl Sidecar {
    /// The lock even after a panic elsewhere: shutting the server down must
    /// still work.
    fn lock(&self) -> MutexGuard<'_, Option<Server>> {
        self.0.lock().unwrap_or_else(|e| e.into_inner())
    }
}

/// The port to serve on: the remembered one when free (see
/// `bind_remembered_port`). This is only a guess: the server reports the
/// port it actually holds (READY_PREFIX).
fn pick_port(app: &AppHandle) -> std::io::Result<u16> {
    Ok(bind_remembered_port(app)?.local_addr()?.port())
}

fn new_token() -> String {
    let mut bytes = [0u8; 32];
    getrandom::fill(&mut bytes).expect("no OS random source");
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

/// Directory of this executable; Tauri puts the sidecars (typstify-server,
/// typst, tinymist) next to it, where the server also looks for typst and
/// tinymist.
fn exe_dir() -> PathBuf {
    std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(Path::to_path_buf))
        .unwrap_or_default()
}

/// Starts the server. The receiver gets the address from its READY line;
/// every stdout line also goes to the log next to stderr.
fn start_server(app: &AppHandle, port: u16, token: &str) -> Result<(Server, Receiver<String>), String> {
    let paths = app.path();
    let resources = paths.resource_dir().map_err(|e| e.to_string())?;
    let logs = paths.app_log_dir().map_err(|e| e.to_string())?;
    let projects = paths
        .document_dir()
        .map(|d| d.join("Typstify"))
        .map_err(|e| e.to_string())?;
    fs::create_dir_all(&logs).map_err(|e| e.to_string())?;
    fs::create_dir_all(&projects).map_err(|e| e.to_string())?;
    let log = File::create(logs.join("typstify-server.log")).map_err(|e| e.to_string())?;
    let mut log_out = log.try_clone().map_err(|e| e.to_string())?;

    let dir = exe_dir();
    let exe = dir.join(if cfg!(windows) { "typstify-server.exe" } else { "typstify-server" });
    let mut cmd = Command::new(&exe);
    cmd.current_dir(&dir)
        .arg("-addr")
        .arg(format!("127.0.0.1:{port}"))
        .arg("-static-dir")
        .arg(resources.join("web"))
        .arg("-project")
        .arg(&projects)
        .env("TYPSTIFY_LAUNCH_TOKEN", token)
        .env_remove("TYPSTIFY_SERVER_PASSWORD")
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(log);
    // The bundled @local/chessbook and its dependency, unless the user
    // points typst somewhere else.
    if std::env::var_os("TYPST_PACKAGE_PATH").is_none() {
        cmd.env("TYPST_PACKAGE_PATH", resources.join("packages"));
    }
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        const CREATE_NO_WINDOW: u32 = 0x0800_0000;
        cmd.creation_flags(CREATE_NO_WINDOW);
    }

    #[cfg(windows)]
    let job = job::KillOnClose::new().map_err(|e| format!("job object: {e}"))?;
    let mut child = cmd
        .spawn()
        .map_err(|e| format!("không chạy được {}: {e}", exe.display()))?;
    #[cfg(windows)]
    if let Err(e) = job.assign(&child) {
        // Not in the job, so nothing would stop it later: stop it now.
        let _ = child.kill();
        let _ = child.wait();
        return Err(format!("job object: {e}"));
    }

    let (ready_tx, ready_rx) = mpsc::channel();
    let stdout = child.stdout.take().expect("piped stdout");
    thread::spawn(move || {
        for line in BufReader::new(stdout).lines().map_while(Result::ok) {
            if let Some(addr) = line.strip_prefix(READY_PREFIX) {
                let _ = ready_tx.send(addr.trim().to_string());
            }
            let _ = writeln!(log_out, "{line}");
        }
    });

    Ok((
        Server {
            child,
            #[cfg(windows)]
            _job: job,
        },
        ready_rx,
    ))
}

/// Waits for the server's READY line and returns the port it holds. Only
/// that line counts, not "something answers on the port": another program
/// that took the port must never receive the window (and its token).
fn wait_ready(app: &AppHandle, ready: &Receiver<String>) -> Result<u16, String> {
    let deadline = Instant::now() + START_TIMEOUT;
    while Instant::now() < deadline {
        match ready.recv_timeout(Duration::from_millis(250)) {
            Ok(addr) => {
                return addr
                    .strip_prefix("127.0.0.1:")
                    .and_then(|p| p.parse().ok())
                    .ok_or_else(|| format!("Máy chủ báo địa chỉ lạ: {addr}"));
            }
            Err(RecvTimeoutError::Disconnected) => return Err("Máy chủ dừng ngay khi khởi động.".into()),
            Err(RecvTimeoutError::Timeout) => {}
        }
        if let Some(server) = app.state::<Sidecar>().lock().as_mut() {
            if let Ok(Some(status)) = server.child.try_wait() {
                return Err(format!("Máy chủ dừng ngay khi khởi động ({status})."));
            }
        }
    }
    Err("Máy chủ khởi động quá lâu.".into())
}

fn show_error(app: &AppHandle, message: &str) {
    if let Some(splash) = app.get_webview_window("splash") {
        let logs = app
            .path()
            .app_log_dir()
            .map(|d| d.join("typstify-server.log").display().to_string())
            .unwrap_or_default();
        let text = format!("{message}\nNhật ký: {logs}");
        if let Ok(literal) = serde_json::to_string(&text) {
            let _ = splash.eval(format!("document.getElementById('status').textContent = {literal}"));
        }
    }
}

fn launch(app: AppHandle) {
    let guess = match pick_port(&app) {
        Ok(p) => p,
        Err(e) => return show_error(&app, &format!("Không tìm được cổng trống: {e}")),
    };
    let token = new_token();
    let ready = match start_server(&app, guess, &token) {
        Ok((server, ready)) => {
            *app.state::<Sidecar>().lock() = Some(server);
            ready
        }
        Err(e) => return show_error(&app, &e),
    };
    let port = match wait_ready(&app, &ready) {
        Ok(p) if p == guess => p,
        Ok(p) => return show_error(&app, &format!("Máy chủ nghe ở cổng {p} thay vì {guess}.")),
        Err(e) => return show_error(&app, &e),
    };

    let url = format!("http://127.0.0.1:{port}/api/auth/launch?token={token}");
    let nav_app = app.clone();
    let popup_app = app.clone();
    let built = url
        .parse()
        .map_err(|e| format!("{e}"))
        .and_then(|u| {
            WebviewWindowBuilder::new(&app, "main", WebviewUrl::External(u))
                .title("Dương Sinh Chess Studio")
                .inner_size(1360.0, 860.0)
                .min_inner_size(800.0, 560.0)
                .center()
                .on_navigation(move |url| {
                    if is_app_url(url, port) {
                        return true;
                    }
                    open_outside(&nav_app, url);
                    false
                })
                .on_new_window(move |url, _features| {
                    // window.open/target=_blank: the app's own pages (the
                    // exported PDF) open in a window of the app, which has
                    // the session; links out go to the system browser.
                    if is_app_url(&url, port) {
                        return NewWindowResponse::Allow;
                    }
                    open_outside(&popup_app, &url);
                    NewWindowResponse::Deny
                })
                .build()
                .map_err(|e| e.to_string())
        });
    match built {
        Ok(_) => {
            if let Some(splash) = app.get_webview_window("splash") {
                let _ = splash.close();
            }
        }
        Err(e) => show_error(&app, &format!("Không mở được cửa sổ: {e}")),
    }
}

fn stop_server(app: &AppHandle) {
    if let Some(mut server) = app.state::<Sidecar>().lock().take() {
        let _ = server.child.kill();
        let _ = server.child.wait();
        // Dropping the job closes it: tinymist and typst go with it.
    }
}

pub fn run() {
    let app = tauri::Builder::default()
        // One instance: the server's database opens only once.
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            let window = app
                .get_webview_window("main")
                .or_else(|| app.get_webview_window("splash"));
            if let Some(w) = window {
                let _ = w.unminimize();
                let _ = w.set_focus();
            }
        }))
        .plugin(tauri_plugin_opener::init())
        .manage(Sidecar(Mutex::new(None)))
        .setup(|app| {
            let handle = app.handle().clone();
            thread::spawn(move || launch(handle));
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building Typstify");

    app.run(|handle, event| {
        if let RunEvent::Exit = event {
            stop_server(handle);
        }
    });
}

#[cfg(windows)]
mod job {
    //! A Windows job object that kills every process in it when closed: the
    //! server and what it starts (tinymist, typst, AI agents) end with the
    //! app, even if the app crashes.
    use std::os::windows::io::AsRawHandle;
    use std::process::Child;

    use windows_sys::Win32::Foundation::{CloseHandle, HANDLE};
    use windows_sys::Win32::System::JobObjects::{
        AssignProcessToJobObject, CreateJobObjectW, JobObjectExtendedLimitInformation,
        SetInformationJobObject, JOBOBJECT_EXTENDED_LIMIT_INFORMATION,
        JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
    };

    pub struct KillOnClose(HANDLE);

    // The handle is only used to assign processes and to close the job.
    unsafe impl Send for KillOnClose {}

    impl KillOnClose {
        pub fn new() -> std::io::Result<Self> {
            unsafe {
                let job = CreateJobObjectW(std::ptr::null(), std::ptr::null());
                if job.is_null() {
                    return Err(std::io::Error::last_os_error());
                }
                let mut info: JOBOBJECT_EXTENDED_LIMIT_INFORMATION = std::mem::zeroed();
                info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
                let ok = SetInformationJobObject(
                    job,
                    JobObjectExtendedLimitInformation,
                    &info as *const _ as *const _,
                    std::mem::size_of::<JOBOBJECT_EXTENDED_LIMIT_INFORMATION>() as u32,
                );
                if ok == 0 {
                    let err = std::io::Error::last_os_error();
                    CloseHandle(job);
                    return Err(err);
                }
                Ok(Self(job))
            }
        }

        pub fn assign(&self, child: &Child) -> std::io::Result<()> {
            let ok = unsafe { AssignProcessToJobObject(self.0, child.as_raw_handle() as HANDLE) };
            if ok == 0 {
                return Err(std::io::Error::last_os_error());
            }
            Ok(())
        }
    }

    impl Drop for KillOnClose {
        fn drop(&mut self) {
            unsafe {
                CloseHandle(self.0);
            }
        }
    }
}
