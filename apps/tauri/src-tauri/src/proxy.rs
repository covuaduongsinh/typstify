//! The mobile app's local proxy (docs/plans/plan_phase7c_tauri_mobile.md).
//!
//! The WebView loads http://127.0.0.1:<port>/. The bundled web UI is served
//! from the app's assets; the server's own paths (/api, /ws, /preview) are
//! forwarded to the self-hosted typstify-server. The UI therefore sees one
//! origin, exactly like the web version: same-origin cookies, WebSockets and
//! the preview iframe work unchanged, and the server needs no CORS.
//!
//! The proxy holds no credentials: the session cookie lives in the WebView's
//! cookie store (for 127.0.0.1:<port>), so another app on the phone calling
//! the port is just an anonymous client of the server. The server can only
//! be chosen through the shell's IPC (src/mobile.rs), never over HTTP.

use std::convert::Infallible;
use std::net::IpAddr;
use std::sync::{Arc, RwLock};
use std::time::Duration;

use bytes::Bytes;
use http_body_util::combinators::BoxBody;
use http_body_util::{BodyExt, Empty, Full};
use hyper::body::Incoming;
use hyper::header::{self, HeaderMap, HeaderValue};
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response, StatusCode};
use hyper_rustls::HttpsConnector;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client;
use hyper_util::rt::{TokioExecutor, TokioIo};
use tokio::net::TcpListener;

pub type Body = BoxBody<Bytes, hyper::Error>;

/// Reads one bundled asset by its path ("index.html", "assets/x.js");
/// `None` when the app has no such file.
pub type Assets = Box<dyn Fn(&str) -> Option<Vec<u8>> + Send + Sync>;

/// Paths served by typstify-server (server/server.go); everything else is
/// the bundled UI.
const REMOTE_PREFIXES: &[&str] = &["/api/", "/ws/", "/preview/"];

/// Paths of the shell itself: the WebView's navigation hook handles them
/// (src/mobile.rs); over HTTP they do not exist.
pub const APP_PREFIX: &str = "/__app/";

const PROBE_TIMEOUT: Duration = Duration::from_secs(10);

/// The server requests go to: scheme and authority only, the server is
/// always mounted at the root.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Upstream {
    scheme: &'static str,
    authority: String,
}

impl Upstream {
    /// Accepts `https://host[:port]`, or plain `http://` for a server on
    /// this device or the local network (a LAN box, the emulator's host at
    /// 10.0.2.2); anything sent over the Internet must be encrypted.
    pub fn parse(input: &str) -> Result<Self, String> {
        let url = url::Url::parse(input.trim()).map_err(|_| "Địa chỉ máy chủ không hợp lệ.".to_string())?;
        let scheme = match url.scheme() {
            "https" => "https",
            "http" => "http",
            _ => return Err("Địa chỉ phải bắt đầu bằng https://".into()),
        };
        if !url.username().is_empty() || url.password().is_some() {
            return Err("Không ghi tên đăng nhập hay mật khẩu trong địa chỉ.".into());
        }
        if !matches!(url.path(), "" | "/") || url.query().is_some() || url.fragment().is_some() {
            return Err("Chỉ nhập địa chỉ gốc của máy chủ, ví dụ https://typst.example.com".into());
        }
        let host = url.host().ok_or("Thiếu tên máy chủ.")?;
        if scheme == "http" && !is_local_host(&host) {
            return Err("Máy chủ trên Internet phải dùng https://".into());
        }
        Ok(Self {
            scheme,
            authority: url.authority().to_string(),
        })
    }

    pub fn origin(&self) -> String {
        format!("{}://{}", self.scheme, self.authority)
    }
}

fn is_local_host(host: &url::Host<&str>) -> bool {
    let ip = match host {
        url::Host::Domain(d) => return d.eq_ignore_ascii_case("localhost"),
        url::Host::Ipv4(ip) => IpAddr::V4(*ip),
        url::Host::Ipv6(ip) => IpAddr::V6(*ip),
    };
    match ip {
        IpAddr::V4(v4) => v4.is_loopback() || v4.is_private() || v4.is_link_local(),
        IpAddr::V6(v6) => {
            v6.is_loopback()
                // Unique local (fc00::/7) and link-local (fe80::/10).
                || (v6.segments()[0] & 0xfe00) == 0xfc00
                || (v6.segments()[0] & 0xffc0) == 0xfe80
        }
    }
}

pub struct Proxy {
    upstream: RwLock<Option<Upstream>>,
    assets: Assets,
    client: Client<HttpsConnector<HttpConnector>, Body>,
}

impl Proxy {
    pub fn new(assets: Assets) -> Self {
        let https = hyper_rustls::HttpsConnectorBuilder::new()
            .with_webpki_roots()
            .https_or_http()
            .enable_http1()
            .build();
        Self {
            upstream: RwLock::new(None),
            assets,
            client: Client::builder(TokioExecutor::new()).build(https),
        }
    }

    pub fn upstream(&self) -> Option<Upstream> {
        self.upstream.read().unwrap_or_else(|e| e.into_inner()).clone()
    }

    pub fn set_upstream(&self, upstream: Option<Upstream>) {
        *self.upstream.write().unwrap_or_else(|e| e.into_inner()) = upstream;
    }

    /// Checks that `upstream` is a typstify-server before it is saved.
    pub async fn probe(&self, upstream: &Upstream) -> Result<(), String> {
        let uri = format!("{}/api/auth/status", upstream.origin());
        let req = Request::get(uri)
            .header(header::ACCEPT, "application/json")
            .body(empty())
            .map_err(|e| e.to_string())?;
        let resp = tokio::time::timeout(PROBE_TIMEOUT, self.client.request(req))
            .await
            .map_err(|_| "Máy chủ không trả lời.".to_string())?
            .map_err(|e| format!("Không kết nối được máy chủ: {e}"))?;
        if resp.status() != StatusCode::OK {
            return Err(format!("Máy chủ trả mã {}.", resp.status().as_u16()));
        }
        let body = tokio::time::timeout(PROBE_TIMEOUT, resp.into_body().collect())
            .await
            .map_err(|_| "Máy chủ không trả lời.".to_string())?
            .map_err(|e| e.to_string())?
            .to_bytes();
        let json: serde_json::Value =
            serde_json::from_slice(&body).map_err(|_| "Đây không phải máy chủ Typstify.".to_string())?;
        if json.get("authRequired").is_none() {
            return Err("Đây không phải máy chủ Typstify.".into());
        }
        Ok(())
    }

    /// `local` is the address the proxy listens on ("127.0.0.1:<port>").
    pub async fn handle(&self, req: Request<Incoming>, local: &str) -> Response<Body> {
        // DNS rebinding: a site opened in another browser on the phone can
        // resolve its own name to 127.0.0.1 and call this port. Its Host and
        // Origin agree, so forwarding would rewrite both to the server's and
        // defeat the server's own Host/Origin checks -- letting the site read
        // the server's anonymous pages, a LAN-only server's included. The
        // WebView always sends the loopback address itself.
        if req.headers().get(header::HOST).and_then(|h| h.to_str().ok()) != Some(local) {
            return status(StatusCode::MISDIRECTED_REQUEST);
        }
        let path = req.uri().path();
        if path.starts_with(APP_PREFIX) {
            return status(StatusCode::NOT_FOUND);
        }
        if path == "/preview" || REMOTE_PREFIXES.iter().any(|p| path.starts_with(p)) {
            return self.forward(req).await;
        }
        let path = path.to_string();
        self.asset(&path)
    }

    fn asset(&self, path: &str) -> Response<Body> {
        let key = path.trim_start_matches('/');
        let key = if key.is_empty() { "index.html" } else { key };
        if key.split('/').any(|seg| seg == ".." || seg == ".") {
            return status(StatusCode::NOT_FOUND);
        }
        let (key, bytes) = match (self.assets)(key) {
            Some(b) => (key, b),
            // A missing file must not turn into the HTML shell (a cached
            // stale chunk would then fail to parse, see server/static.go);
            // other paths are routes of the single-page app.
            None if key.rsplit('/').next().is_some_and(|last| last.contains('.')) => {
                return status(StatusCode::NOT_FOUND)
            }
            None => match (self.assets)("index.html") {
                Some(b) => ("index.html", b),
                None => return status(StatusCode::NOT_FOUND),
            },
        };
        let cache = if key.starts_with("assets/") {
            "public, max-age=31536000, immutable"
        } else {
            "no-cache"
        };
        Response::builder()
            .header(header::CONTENT_TYPE, mime_type(key))
            .header(header::CACHE_CONTROL, cache)
            .header(header::X_CONTENT_TYPE_OPTIONS, "nosniff")
            .body(full(bytes))
            .unwrap_or_else(|_| status(StatusCode::INTERNAL_SERVER_ERROR))
    }

    async fn forward(&self, req: Request<Incoming>) -> Response<Body> {
        let Some(up) = self.upstream() else {
            return json(StatusCode::SERVICE_UNAVAILABLE, r#"{"error":"Chưa chọn máy chủ."}"#);
        };
        let (mut parts, body) = req.into_parts();
        let local = parts
            .headers
            .get(header::HOST)
            .and_then(|h| h.to_str().ok())
            .map(|h| format!("http://{h}"));
        let downstream_upgrade = parts.extensions.remove::<hyper::upgrade::OnUpgrade>();
        let path = parts.uri.path_and_query().map_or("/", |p| p.as_str());
        parts.uri = match format!("{}{path}", up.origin()).parse() {
            Ok(u) => u,
            Err(_) => return status(StatusCode::BAD_REQUEST),
        };
        rewrite_request_headers(&mut parts.headers, &up, local.as_deref());

        let mut resp = match self.client.request(Request::from_parts(parts, body.boxed())).await {
            Ok(r) => r,
            Err(_) => return json(StatusCode::BAD_GATEWAY, r#"{"error":"Không kết nối được máy chủ."}"#),
        };

        if resp.status() == StatusCode::SWITCHING_PROTOCOLS {
            // WebSocket: once both sides have switched, pipe the bytes.
            let upstream_upgrade = hyper::upgrade::on(&mut resp);
            if let Some(downstream) = downstream_upgrade {
                tokio::spawn(async move {
                    if let (Ok(down), Ok(up)) = tokio::join!(downstream, upstream_upgrade) {
                        let _ = tokio::io::copy_bidirectional(&mut TokioIo::new(down), &mut TokioIo::new(up)).await;
                    }
                });
            }
            let (parts, _) = resp.into_parts();
            return Response::from_parts(parts, empty());
        }

        rewrite_response_headers(resp.headers_mut(), &up);
        resp.map(BodyExt::boxed)
    }
}

/// The server checks Host and, for writes and WebSockets, that Origin
/// matches it: requests from the app's own page look like they came from a
/// page of the server. A foreign Origin is passed on unchanged, so the
/// server still refuses it.
fn rewrite_request_headers(headers: &mut HeaderMap, up: &Upstream, local: Option<&str>) {
    let origin = up.origin();
    if let Ok(host) = HeaderValue::from_str(&up.authority) {
        headers.insert(header::HOST, host);
    }
    if let (Some(local), Some(value)) = (local, headers.get(header::ORIGIN)) {
        if value.as_bytes() == local.as_bytes() {
            if let Ok(v) = HeaderValue::from_str(&origin) {
                headers.insert(header::ORIGIN, v);
            }
        }
    }
    if let Some(referer) = headers.get(header::REFERER).and_then(|v| v.to_str().ok()) {
        let rewritten = local
            .and_then(|l| referer.strip_prefix(l))
            .filter(|rest| rest.is_empty() || rest.starts_with('/'))
            .and_then(|rest| HeaderValue::from_str(&format!("{origin}{rest}")).ok());
        match rewritten {
            Some(v) => {
                headers.insert(header::REFERER, v);
            }
            None => {
                headers.remove(header::REFERER);
            }
        }
    }
}

fn rewrite_response_headers(headers: &mut HeaderMap, up: &Upstream) {
    let cookies: Vec<HeaderValue> = headers
        .get_all(header::SET_COOKIE)
        .iter()
        .filter_map(|v| v.to_str().ok())
        .filter_map(|v| HeaderValue::from_str(&local_cookie(v)).ok())
        .collect();
    headers.remove(header::SET_COOKIE);
    for c in cookies {
        headers.append(header::SET_COOKIE, c);
    }
    let origin = up.origin();
    if let Some(location) = headers.get(header::LOCATION).and_then(|v| v.to_str().ok()) {
        if let Some(rest) = location.strip_prefix(&origin) {
            let rest = if rest.is_empty() { "/" } else { rest };
            if let Ok(v) = HeaderValue::from_str(rest) {
                headers.insert(header::LOCATION, v);
            }
        }
    }
    // Meant for the server's host name, not for 127.0.0.1.
    headers.remove(header::STRICT_TRANSPORT_SECURITY);
    headers.remove(header::ALT_SVC);
}

/// A cookie of the server, made to stick to 127.0.0.1:<port>: no Domain,
/// and no Secure (the page is plain http on the loopback; the hop to the
/// server is still TLS).
fn local_cookie(cookie: &str) -> String {
    let mut parts = cookie.split(';').map(str::trim).filter(|p| !p.is_empty());
    let mut out = vec![parts.next().unwrap_or_default().to_string()];
    for attr in parts {
        let name = attr.split('=').next().unwrap_or_default().trim().to_ascii_lowercase();
        match name.as_str() {
            "domain" | "secure" | "partitioned" => {}
            // SameSite=None needs Secure; the page and the API share an
            // origin anyway.
            "samesite" if attr.to_ascii_lowercase().ends_with("none") => out.push("SameSite=Lax".into()),
            _ => out.push(attr.to_string()),
        }
    }
    out.join("; ")
}

fn mime_type(path: &str) -> &'static str {
    let ext = path.rsplit('.').next().unwrap_or_default().to_ascii_lowercase();
    match ext.as_str() {
        "html" => "text/html; charset=utf-8",
        "js" | "mjs" => "text/javascript; charset=utf-8",
        "css" => "text/css; charset=utf-8",
        "json" => "application/json",
        "webmanifest" => "application/manifest+json",
        "wasm" => "application/wasm",
        "svg" => "image/svg+xml",
        "png" => "image/png",
        "jpg" | "jpeg" => "image/jpeg",
        "webp" => "image/webp",
        "ico" => "image/x-icon",
        "woff2" => "font/woff2",
        "woff" => "font/woff",
        "ttf" => "font/ttf",
        "otf" => "font/otf",
        "txt" => "text/plain; charset=utf-8",
        _ => "application/octet-stream",
    }
}

fn full(bytes: impl Into<Bytes>) -> Body {
    Full::new(bytes.into()).map_err(|never| match never {}).boxed()
}

fn empty() -> Body {
    Empty::<Bytes>::new().map_err(|never| match never {}).boxed()
}

fn status(code: StatusCode) -> Response<Body> {
    let mut r = Response::new(empty());
    *r.status_mut() = code;
    r
}

fn json(code: StatusCode, body: &'static str) -> Response<Body> {
    let mut r = Response::new(full(body));
    *r.status_mut() = code;
    r.headers_mut()
        .insert(header::CONTENT_TYPE, HeaderValue::from_static("application/json"));
    r
}

/// Serves the proxy on `listener` until the app exits.
pub async fn serve(listener: TcpListener, proxy: Arc<Proxy>) {
    let local: Arc<str> = match listener.local_addr() {
        Ok(a) => a.to_string().into(),
        Err(_) => return,
    };
    loop {
        let stream = match listener.accept().await {
            Ok((s, _)) => s,
            Err(_) => {
                tokio::time::sleep(Duration::from_millis(100)).await;
                continue;
            }
        };
        let proxy = proxy.clone();
        let local = local.clone();
        tokio::spawn(async move {
            let svc = service_fn(move |req| {
                let proxy = proxy.clone();
                let local = local.clone();
                async move { Ok::<_, Infallible>(proxy.handle(req, &local).await) }
            });
            let _ = http1::Builder::new()
                .serve_connection(TokioIo::new(stream), svc)
                .with_upgrades()
                .await;
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};
    use tokio::net::TcpStream;

    #[test]
    fn upstream_rules() {
        let ok = |s: &str| Upstream::parse(s).map(|u| u.origin());
        assert_eq!(ok("https://typst.example.com").unwrap(), "https://typst.example.com");
        assert_eq!(ok(" https://typst.example.com:8443/ ").unwrap(), "https://typst.example.com:8443");
        assert_eq!(ok("https://typst.example.com:443").unwrap(), "https://typst.example.com");
        assert_eq!(ok("http://10.0.2.2:8080").unwrap(), "http://10.0.2.2:8080");
        assert_eq!(ok("http://192.168.1.20").unwrap(), "http://192.168.1.20");
        assert_eq!(ok("http://[::1]:8080").unwrap(), "http://[::1]:8080");
        assert!(ok("http://typst.example.com").is_err(), "plain http over the Internet");
        assert!(ok("http://8.8.8.8").is_err());
        assert!(ok("https://u:p@typst.example.com").is_err());
        assert!(ok("https://typst.example.com/sub").is_err());
        assert!(ok("https://typst.example.com/?x=1").is_err());
        assert!(ok("ftp://typst.example.com").is_err());
        assert!(ok("typst.example.com").is_err());
    }

    #[test]
    fn cookies_stick_to_the_loopback() {
        assert_eq!(
            local_cookie("typstify_session=abc; Path=/; Domain=example.com; HttpOnly; Secure; SameSite=Strict"),
            "typstify_session=abc; Path=/; HttpOnly; SameSite=Strict"
        );
        assert_eq!(local_cookie("a=b; SameSite=None; Secure"), "a=b; SameSite=Lax");
    }

    #[test]
    fn foreign_origins_pass_unchanged() {
        let up = Upstream::parse("https://s.example.com").unwrap();
        let mut h = HeaderMap::new();
        h.insert(header::HOST, HeaderValue::from_static("127.0.0.1:5000"));
        h.insert(header::ORIGIN, HeaderValue::from_static("http://evil.example"));
        h.insert(header::REFERER, HeaderValue::from_static("http://evil.example/x"));
        rewrite_request_headers(&mut h, &up, Some("http://127.0.0.1:5000"));
        assert_eq!(h[header::HOST], "s.example.com");
        assert_eq!(h[header::ORIGIN], "http://evil.example");
        assert!(h.get(header::REFERER).is_none());

        let mut h = HeaderMap::new();
        h.insert(header::ORIGIN, HeaderValue::from_static("http://127.0.0.1:5000"));
        h.insert(header::REFERER, HeaderValue::from_static("http://127.0.0.1:5000/p?q=1"));
        rewrite_request_headers(&mut h, &up, Some("http://127.0.0.1:5000"));
        assert_eq!(h[header::ORIGIN], "https://s.example.com");
        assert_eq!(h[header::REFERER], "https://s.example.com/p?q=1");
    }

    /// A fake typstify-server: echoes what it received; /ws/echo switches
    /// protocols and echoes bytes.
    async fn fake_server() -> String {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(async move {
            loop {
                let (stream, _) = listener.accept().await.unwrap();
                tokio::spawn(async move {
                    let svc = service_fn(|mut req: Request<Incoming>| async move {
                        if req.uri().path() == "/ws/echo" {
                            let on = hyper::upgrade::on(&mut req);
                            tokio::spawn(async move {
                                let mut io = TokioIo::new(on.await.unwrap());
                                let mut buf = [0u8; 5];
                                io.read_exact(&mut buf).await.unwrap();
                                io.write_all(&buf).await.unwrap();
                            });
                            return Ok::<_, Infallible>(
                                Response::builder()
                                    .status(StatusCode::SWITCHING_PROTOCOLS)
                                    .header(header::CONNECTION, "upgrade")
                                    .header(header::UPGRADE, "websocket")
                                    .body(empty())
                                    .unwrap(),
                            );
                        }
                        let seen = format!(
                            "{} {} host={} origin={}",
                            req.method(),
                            req.uri(),
                            req.headers()[header::HOST].to_str().unwrap(),
                            req.headers().get(header::ORIGIN).map_or("-", |v| v.to_str().unwrap()),
                        );
                        let body = req.into_body().collect().await.unwrap().to_bytes();
                        Ok(Response::builder()
                            .header(header::SET_COOKIE, "s=1; Path=/; Secure; HttpOnly")
                            .header(header::LOCATION, format!("http://{addr}/next"))
                            .body(full(format!("{seen} body={}", String::from_utf8_lossy(&body))))
                            .unwrap())
                    });
                    let _ = http1::Builder::new()
                        .serve_connection(TokioIo::new(stream), svc)
                        .with_upgrades()
                        .await;
                });
            }
        });
        format!("http://{addr}")
    }

    async fn start_proxy(upstream: Option<&str>) -> String {
        let files: HashMap<&str, &[u8]> = HashMap::from([
            ("index.html", &b"<!doctype html>shell"[..]),
            ("assets/app.js", &b"console.log(1)"[..]),
        ]);
        let proxy = Arc::new(Proxy::new(Box::new(move |p| files.get(p).map(|b| b.to_vec()))));
        proxy.set_upstream(upstream.map(|u| Upstream::parse(u).unwrap()));
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        tokio::spawn(serve(listener, proxy));
        format!("127.0.0.1:{}", addr.port())
    }

    /// One raw HTTP/1.1 exchange; returns the whole response text.
    async fn raw(addr: &str, request: String) -> String {
        let mut s = TcpStream::connect(addr).await.unwrap();
        s.write_all(request.as_bytes()).await.unwrap();
        let mut out = Vec::new();
        s.read_to_end(&mut out).await.unwrap();
        String::from_utf8_lossy(&out).into_owned()
    }

    #[tokio::test]
    async fn serves_bundled_ui() {
        let proxy = start_proxy(None).await;
        let get = |p: &str| format!("GET {p} HTTP/1.1\r\nHost: {proxy}\r\nConnection: close\r\n\r\n");
        let r = raw(&proxy, get("/")).await;
        assert!(r.starts_with("HTTP/1.1 200") && r.contains("text/html") && r.ends_with("shell"), "{r}");
        let r = raw(&proxy, get("/assets/app.js")).await;
        assert!(r.contains("text/javascript") && r.contains("immutable"), "{r}");
        // A route of the app gets the shell, a missing file does not.
        assert!(raw(&proxy, get("/projects/abc")).await.ends_with("shell"));
        assert!(raw(&proxy, get("/assets/gone.js")).await.starts_with("HTTP/1.1 404"));
        assert!(raw(&proxy, get("/__app/server")).await.starts_with("HTTP/1.1 404"));
        // No server chosen yet.
        assert!(raw(&proxy, get("/api/auth/status")).await.starts_with("HTTP/1.1 503"));
    }

    #[tokio::test]
    async fn refuses_rebound_host_names() {
        let server = fake_server().await;
        let proxy = start_proxy(Some(&server)).await;
        let port = proxy.rsplit(':').next().unwrap();
        let evil = format!("evil.example:{port}");
        let r = raw(
            &proxy,
            format!("GET /api/auth/status HTTP/1.1\r\nHost: {evil}\r\nOrigin: http://{evil}\r\nConnection: close\r\n\r\n"),
        )
        .await;
        assert!(r.starts_with("HTTP/1.1 421"), "{r}");
        let r = raw(&proxy, format!("GET / HTTP/1.1\r\nHost: localhost:{port}\r\nConnection: close\r\n\r\n")).await;
        assert!(r.starts_with("HTTP/1.1 421"), "{r}");
    }

    #[tokio::test]
    async fn forwards_to_the_server() {
        let server = fake_server().await;
        let server_authority = server.trim_start_matches("http://").to_string();
        let proxy = start_proxy(Some(&server)).await;
        let r = raw(
            &proxy,
            format!(
                "POST /api/x?y=1 HTTP/1.1\r\nHost: {proxy}\r\nOrigin: http://{proxy}\r\nContent-Length: 2\r\nConnection: close\r\n\r\nhi"
            ),
        )
        .await;
        assert!(r.contains(&format!("POST /api/x?y=1 host={server_authority} origin={server} body=hi")), "{r}");
        assert!(r.to_ascii_lowercase().contains("set-cookie: s=1; path=/; httponly\r\n"), "{r}");
        assert!(r.to_ascii_lowercase().contains("location: /next\r\n"), "{r}");
    }

    #[tokio::test]
    async fn pipes_websockets() {
        let server = fake_server().await;
        let proxy = start_proxy(Some(&server)).await;
        let mut s = TcpStream::connect(&proxy).await.unwrap();
        s.write_all(
            format!(
                "GET /ws/echo HTTP/1.1\r\nHost: {proxy}\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n"
            )
            .as_bytes(),
        )
        .await
        .unwrap();
        let mut head = Vec::new();
        while !head.ends_with(b"\r\n\r\n") {
            let mut b = [0u8; 1];
            s.read_exact(&mut b).await.unwrap();
            head.push(b[0]);
        }
        assert!(String::from_utf8_lossy(&head).starts_with("HTTP/1.1 101"));
        s.write_all(b"hello").await.unwrap();
        let mut echo = [0u8; 5];
        tokio::time::timeout(Duration::from_secs(5), s.read_exact(&mut echo))
            .await
            .unwrap()
            .unwrap();
        assert_eq!(&echo, b"hello");
    }
}
