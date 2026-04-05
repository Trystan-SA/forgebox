//! OAuth 2.0 / OIDC authentication for ForgeBox desktop.
//!
//! Supports Google, Microsoft, and generic OIDC providers. The flow opens the
//! system browser for login, receives the callback on a temporary local HTTP
//! server, exchanges the auth code for tokens, and persists them in the OS
//! keychain via the `keyring` crate.

use anyhow::{anyhow, Context, Result};
use keyring::Entry;
use oauth2::basic::BasicClient;
use oauth2::reqwest::async_http_client;
use oauth2::{
    AuthUrl, AuthorizationCode, ClientId, CsrfToken, PkceCodeChallenge, PkceCodeVerifier,
    RedirectUrl, Scope, TokenResponse, TokenUrl,
};
use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader, Write};
use std::net::TcpListener;
use url::Url;

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const GOOGLE_AUTH_URL: &str = "https://accounts.google.com/o/oauth2/v2/auth";
const GOOGLE_TOKEN_URL: &str = "https://oauth2.googleapis.com/token";

const MICROSOFT_AUTH_URL: &str =
    "https://login.microsoftonline.com/common/oauth2/v2/authorize";
const MICROSOFT_TOKEN_URL: &str =
    "https://login.microsoftonline.com/common/oauth2/v2/token";

const KEYRING_SERVICE: &str = "forgebox-desktop";
const KEYRING_ACCESS_TOKEN: &str = "access_token";
const KEYRING_REFRESH_TOKEN: &str = "refresh_token";
const KEYRING_USER_EMAIL: &str = "user_email";
const KEYRING_USER_NAME: &str = "user_name";
const KEYRING_PROVIDER: &str = "provider";
const KEYRING_GATEWAY_URL: &str = "gateway_url";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/// Persisted authentication state.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthState {
    pub gateway_url: String,
    pub access_token: String,
    pub refresh_token: Option<String>,
    pub user_email: String,
    pub user_name: String,
    pub provider: String,
}

/// Minimal user info returned to the frontend after authentication.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub email: String,
    pub name: String,
}

// ---------------------------------------------------------------------------
// OAuth flow
// ---------------------------------------------------------------------------

/// Run the full OAuth 2.0 authorization-code flow with PKCE.
///
/// 1. Build the authorization URL for the requested provider.
/// 2. Open the system browser so the user can log in.
/// 3. Listen on a random local port for the redirect callback.
/// 4. Exchange the authorization code for tokens.
/// 5. Fetch basic user-info from the provider.
/// 6. Store everything in the OS keychain.
pub async fn start_oauth_flow(provider: &str, gateway_url: &str) -> Result<AuthState> {
    let (auth_url, token_url, scopes, client_id) = provider_endpoints(provider, gateway_url)?;

    let listener = TcpListener::bind("127.0.0.1:0").context("failed to bind callback listener")?;
    let port = listener.local_addr()?.port();
    let redirect_uri = format!("http://127.0.0.1:{port}/callback");

    let client = BasicClient::new(
        ClientId::new(client_id),
        None,
        AuthUrl::new(auth_url).context("invalid auth URL")?,
        Some(TokenUrl::new(token_url).context("invalid token URL")?),
    )
    .set_redirect_uri(RedirectUrl::new(redirect_uri).context("invalid redirect URL")?);

    let (pkce_challenge, pkce_verifier) = PkceCodeChallenge::new_random_sha256();

    let (authorize_url, csrf_state) = {
        let mut req = client.authorize_url(CsrfToken::new_random);
        for scope in &scopes {
            req = req.add_scope(Scope::new(scope.clone()));
        }
        req.set_pkce_challenge(pkce_challenge).url()
    };

    open::that(authorize_url.as_str()).context("failed to open system browser")?;

    let (code, returned_state) = wait_for_callback(listener)?;

    if returned_state != *csrf_state.secret() {
        return Err(anyhow!("CSRF state mismatch — possible attack"));
    }

    let token_result = client
        .exchange_code(AuthorizationCode::new(code))
        .set_pkce_verifier(pkce_verifier)
        .request_async(async_http_client)
        .await
        .context("token exchange failed")?;

    let access_token = token_result.access_token().secret().clone();
    let refresh_token = token_result.refresh_token().map(|t| t.secret().clone());

    let user_info = fetch_user_info(provider, &access_token, gateway_url).await?;

    let state = AuthState {
        gateway_url: gateway_url.to_string(),
        access_token,
        refresh_token,
        user_email: user_info.email,
        user_name: user_info.name,
        provider: provider.to_string(),
    };

    store_auth(&state)?;
    Ok(state)
}

/// Load previously-stored authentication from the OS keychain.
pub fn load_stored_auth() -> Option<AuthState> {
    let get = |key: &str| -> Option<String> {
        Entry::new(KEYRING_SERVICE, key).ok()?.get_password().ok()
    };

    Some(AuthState {
        gateway_url: get(KEYRING_GATEWAY_URL)?,
        access_token: get(KEYRING_ACCESS_TOKEN)?,
        refresh_token: get(KEYRING_REFRESH_TOKEN),
        user_email: get(KEYRING_USER_EMAIL)?,
        user_name: get(KEYRING_USER_NAME)?,
        provider: get(KEYRING_PROVIDER)?,
    })
}

/// Remove all stored credentials from the OS keychain.
pub fn clear_auth() {
    let keys = [
        KEYRING_ACCESS_TOKEN,
        KEYRING_REFRESH_TOKEN,
        KEYRING_USER_EMAIL,
        KEYRING_USER_NAME,
        KEYRING_PROVIDER,
        KEYRING_GATEWAY_URL,
    ];
    for key in keys {
        if let Ok(entry) = Entry::new(KEYRING_SERVICE, key) {
            let _ = entry.delete_credential();
        }
    }
}

/// Refresh the access token using the stored refresh token.
///
/// Returns the new access token on success.
pub async fn refresh_token_if_needed(state: &AuthState) -> Result<String> {
    let refresh_token = state
        .refresh_token
        .as_deref()
        .ok_or_else(|| anyhow!("no refresh token available"))?;

    let (auth_url, token_url, _, client_id) =
        provider_endpoints(&state.provider, &state.gateway_url)?;

    let client = BasicClient::new(
        ClientId::new(client_id),
        None,
        AuthUrl::new(auth_url).context("invalid auth URL")?,
        Some(TokenUrl::new(token_url).context("invalid token URL")?),
    );

    let token_result = client
        .exchange_refresh_token(&oauth2::RefreshToken::new(refresh_token.to_string()))
        .request_async(async_http_client)
        .await
        .context("refresh token exchange failed")?;

    let new_access_token = token_result.access_token().secret().clone();

    // Persist the new access token.
    if let Ok(entry) = Entry::new(KEYRING_SERVICE, KEYRING_ACCESS_TOKEN) {
        let _ = entry.set_password(&new_access_token);
    }

    Ok(new_access_token)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

/// Return (auth_url, token_url, scopes, client_id) for the given provider.
fn provider_endpoints(
    provider: &str,
    gateway_url: &str,
) -> Result<(String, String, Vec<String>, String)> {
    match provider {
        "google" => Ok((
            GOOGLE_AUTH_URL.to_string(),
            GOOGLE_TOKEN_URL.to_string(),
            vec![
                "openid".into(),
                "email".into(),
                "profile".into(),
            ],
            // In production the client ID is fetched from the gateway's
            // well-known endpoint; for now use an env-var fallback.
            std::env::var("FORGEBOX_GOOGLE_CLIENT_ID")
                .unwrap_or_else(|_| "forgebox-desktop".into()),
        )),
        "microsoft" => Ok((
            MICROSOFT_AUTH_URL.to_string(),
            MICROSOFT_TOKEN_URL.to_string(),
            vec![
                "openid".into(),
                "email".into(),
                "profile".into(),
                "User.Read".into(),
            ],
            std::env::var("FORGEBOX_MICROSOFT_CLIENT_ID")
                .unwrap_or_else(|_| "forgebox-desktop".into()),
        )),
        "oidc" => {
            // Generic OIDC — the gateway exposes a discovery document.
            let base = gateway_url.trim_end_matches('/');
            Ok((
                format!("{base}/auth/oidc/authorize"),
                format!("{base}/auth/oidc/token"),
                vec!["openid".into(), "email".into(), "profile".into()],
                std::env::var("FORGEBOX_OIDC_CLIENT_ID")
                    .unwrap_or_else(|_| "forgebox-desktop".into()),
            ))
        }
        _ => Err(anyhow!("unsupported OAuth provider: {provider}")),
    }
}

/// Block until the OAuth redirect callback arrives on `listener`.
///
/// Returns `(code, state)` extracted from the query string.
fn wait_for_callback(listener: TcpListener) -> Result<(String, String)> {
    let (mut stream, _) = listener.accept().context("failed to accept callback connection")?;

    let mut reader = BufReader::new(&stream);
    let mut request_line = String::new();
    reader
        .read_line(&mut request_line)
        .context("failed to read callback request")?;

    // Parse the GET request line: "GET /callback?code=...&state=... HTTP/1.1"
    let url_str = request_line
        .split_whitespace()
        .nth(1)
        .ok_or_else(|| anyhow!("malformed HTTP request line"))?;

    let full_url = Url::parse(&format!("http://127.0.0.1{url_str}"))
        .context("failed to parse callback URL")?;

    let code = full_url
        .query_pairs()
        .find(|(k, _)| k == "code")
        .map(|(_, v)| v.into_owned())
        .ok_or_else(|| anyhow!("no authorization code in callback"))?;

    let state = full_url
        .query_pairs()
        .find(|(k, _)| k == "state")
        .map(|(_, v)| v.into_owned())
        .ok_or_else(|| anyhow!("no state parameter in callback"))?;

    // Send a minimal success page back to the browser.
    let response = "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n\
        <html><body><h2>Authentication successful</h2>\
        <p>You can close this tab and return to ForgeBox.</p></body></html>";
    let _ = stream.write_all(response.as_bytes());

    Ok((code, state))
}

/// Persist authentication state to the OS keychain.
fn store_auth(state: &AuthState) -> Result<()> {
    let set = |key: &str, value: &str| -> Result<()> {
        Entry::new(KEYRING_SERVICE, key)
            .context("failed to create keyring entry")?
            .set_password(value)
            .context("failed to store credential")?;
        Ok(())
    };

    set(KEYRING_ACCESS_TOKEN, &state.access_token)?;
    set(KEYRING_USER_EMAIL, &state.user_email)?;
    set(KEYRING_USER_NAME, &state.user_name)?;
    set(KEYRING_PROVIDER, &state.provider)?;
    set(KEYRING_GATEWAY_URL, &state.gateway_url)?;

    if let Some(ref rt) = state.refresh_token {
        set(KEYRING_REFRESH_TOKEN, rt)?;
    }

    Ok(())
}

/// Fetch user info (email + display name) from the provider's userinfo
/// endpoint or from the ForgeBox gateway.
async fn fetch_user_info(provider: &str, access_token: &str, gateway_url: &str) -> Result<UserInfo> {
    let client = reqwest::Client::new();

    let url = match provider {
        "google" => "https://www.googleapis.com/oauth2/v3/userinfo".to_string(),
        "microsoft" => "https://graph.microsoft.com/v1.0/me".to_string(),
        _ => {
            let base = gateway_url.trim_end_matches('/');
            format!("{base}/auth/userinfo")
        }
    };

    let resp: serde_json::Value = client
        .get(&url)
        .bearer_auth(access_token)
        .send()
        .await
        .context("userinfo request failed")?
        .json()
        .await
        .context("failed to parse userinfo response")?;

    let email = resp
        .get("email")
        .or_else(|| resp.get("mail"))
        .or_else(|| resp.get("userPrincipalName"))
        .and_then(|v| v.as_str())
        .unwrap_or("unknown@example.com")
        .to_string();

    let name = resp
        .get("name")
        .or_else(|| resp.get("displayName"))
        .and_then(|v| v.as_str())
        .unwrap_or("ForgeBox User")
        .to_string();

    Ok(UserInfo { email, name })
}
