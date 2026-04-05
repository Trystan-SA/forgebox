import { Box, LogOut, Globe } from "lucide-react";
import type { UseAuthReturn } from "../hooks/useAuth";

interface DashboardShellProps {
  auth: UseAuthReturn;
}

export function DashboardShell({ auth }: DashboardShellProps) {
  const info = auth.auth;

  return (
    <div className="flex h-screen flex-col">
      {/* Top bar */}
      <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-2.5 shadow-sm">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-forge-600 text-white">
            <Box className="h-4.5 w-4.5" />
          </div>
          <span className="text-sm font-semibold text-gray-900">ForgeBox</span>
        </div>

        <div className="flex items-center gap-4">
          {/* Connected server */}
          {info?.gateway_url && (
            <div className="flex items-center gap-1.5 text-xs text-gray-500">
              <Globe className="h-3.5 w-3.5" />
              <span className="max-w-[200px] truncate">{info.gateway_url}</span>
            </div>
          )}

          {/* User info */}
          {info && (
            <div className="text-right text-xs leading-tight">
              <p className="font-medium text-gray-900">{info.name}</p>
              <p className="text-gray-500">{info.email}</p>
            </div>
          )}

          {/* Disconnect */}
          <button
            type="button"
            onClick={() => auth.logout()}
            className="btn-secondary gap-1.5 px-3 py-1.5 text-xs"
            title="Disconnect"
          >
            <LogOut className="h-3.5 w-3.5" />
            Disconnect
          </button>
        </div>
      </header>

      {/* Main content area -- iframe to the gateway dashboard */}
      <main className="flex-1 overflow-hidden">
        {info?.gateway_url ? (
          <iframe
            src={info.gateway_url}
            title="ForgeBox Dashboard"
            className="h-full w-full border-0"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-gray-500">
            Not connected to a gateway.
          </div>
        )}
      </main>
    </div>
  );
}
