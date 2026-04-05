import { useState } from "react";
import { Box } from "lucide-react";
import type { UseAuthReturn } from "../hooks/useAuth";
import { AuthButton } from "../components/AuthButton";

interface LoginPageProps {
  auth: UseAuthReturn;
}

export function LoginPage({ auth }: LoginPageProps) {
  const [gatewayUrl, setGatewayUrl] = useState("");
  const [remember, setRemember] = useState(true);
  const [activeProvider, setActiveProvider] = useState<
    "google" | "microsoft" | "sso" | null
  >(null);

  const handleLogin = async (provider: "google" | "microsoft" | "sso") => {
    if (!gatewayUrl.trim()) return;
    setActiveProvider(provider);
    try {
      await auth.login(provider, gatewayUrl.trim());
    } finally {
      setActiveProvider(null);
    }
  };

  const isLoading = auth.loading || activeProvider !== null;
  const urlMissing = gatewayUrl.trim().length === 0;

  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* Logo & title */}
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-forge-600 text-white shadow-lg">
            <Box className="h-8 w-8" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-gray-900">
            ForgeBox
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Connect to your company's ForgeBox gateway
          </p>
        </div>

        {/* Card */}
        <div className="card p-6">
          {/* Gateway URL */}
          <label
            htmlFor="gateway-url"
            className="mb-1.5 block text-sm font-medium text-gray-700"
          >
            Gateway URL
          </label>
          <input
            id="gateway-url"
            type="url"
            value={gatewayUrl}
            onChange={(e) => setGatewayUrl(e.target.value)}
            placeholder="https://forgebox.yourcompany.com"
            className="input mb-4"
            autoFocus
          />

          {/* Auth buttons */}
          <div className="space-y-3">
            <AuthButton
              provider="google"
              onClick={() => handleLogin("google")}
              loading={activeProvider === "google"}
              disabled={isLoading || urlMissing}
            />
            <AuthButton
              provider="microsoft"
              onClick={() => handleLogin("microsoft")}
              loading={activeProvider === "microsoft"}
              disabled={isLoading || urlMissing}
            />
            <AuthButton
              provider="sso"
              onClick={() => handleLogin("sso")}
              loading={activeProvider === "sso"}
              disabled={isLoading || urlMissing}
            />
          </div>

          {/* Remember checkbox */}
          <label className="mt-4 flex items-center gap-2 text-sm text-gray-600">
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-forge-600 focus:ring-forge-500"
            />
            Remember this server
          </label>

          {/* Error display */}
          {auth.error && (
            <div className="mt-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
              {auth.error}
            </div>
          )}
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-xs text-gray-400">
          ForgeBox Desktop v0.1.0
        </p>
      </div>
    </div>
  );
}
