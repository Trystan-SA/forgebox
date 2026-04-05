import { useCallback, useEffect, useState } from "react";
import {
  type AuthInfo,
  getAuthState,
  startOauth,
  disconnect as tauriDisconnect,
} from "../lib/tauri";

export interface UseAuthReturn {
  auth: AuthInfo | null;
  loading: boolean;
  error: string | null;
  login: (provider: "google" | "microsoft" | "sso", gatewayUrl: string) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: boolean;
}

export function useAuth(): UseAuthReturn {
  const [auth, setAuth] = useState<AuthInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Hydrate auth state from the Tauri backend on mount.
  useEffect(() => {
    let cancelled = false;
    getAuthState()
      .then((info) => {
        if (!cancelled) setAuth(info);
      })
      .catch((err) => {
        if (!cancelled) setError(String(err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(
    async (provider: "google" | "microsoft" | "sso", gatewayUrl: string) => {
      setLoading(true);
      setError(null);
      try {
        const info = await startOauth(provider, gatewayUrl);
        setAuth(info);
      } catch (err) {
        setError(String(err));
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const logout = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await tauriDisconnect();
      setAuth(null);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    auth,
    loading,
    error,
    login,
    logout,
    isAuthenticated: auth !== null,
  };
}
