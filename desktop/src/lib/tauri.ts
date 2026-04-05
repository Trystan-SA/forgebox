import { invoke } from "@tauri-apps/api/core";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AuthInfo {
  email: string;
  name: string;
  provider: "google" | "microsoft" | "sso";
  gateway_url: string;
}

export interface AppSettings {
  gateway_url: string;
  auto_connect: boolean;
  theme: "light" | "dark" | "system";
  default_provider: "google" | "microsoft" | "sso";
}

// ---------------------------------------------------------------------------
// Tauri command wrappers
// ---------------------------------------------------------------------------

/** Connect to a ForgeBox gateway instance. */
export function connect(gatewayUrl: string): Promise<AuthInfo> {
  return invoke<AuthInfo>("connect", { gatewayUrl });
}

/** Disconnect from the current gateway and clear local session. */
export function disconnect(): Promise<void> {
  return invoke<void>("disconnect");
}

/** Retrieve the persisted authentication state, or null if not logged in. */
export function getAuthState(): Promise<AuthInfo | null> {
  return invoke<AuthInfo | null>("get_auth_state");
}

/** Launch the OAuth / SSO flow for the given provider and gateway. */
export function startOauth(
  provider: "google" | "microsoft" | "sso",
  gatewayUrl: string,
): Promise<AuthInfo> {
  return invoke<AuthInfo>("start_oauth", { provider, gatewayUrl });
}

/** Persist application settings to the Tauri store. */
export function saveSettings(settings: AppSettings): Promise<void> {
  return invoke<void>("save_settings", { settings });
}

/** Load application settings from the Tauri store. */
export function getSettings(): Promise<AppSettings> {
  return invoke<AppSettings>("get_settings");
}
