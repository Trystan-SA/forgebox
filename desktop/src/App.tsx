import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./hooks/useAuth";
import { LoginPage } from "./pages/LoginPage";
import { DashboardShell } from "./pages/DashboardShell";
import { Loader2 } from "lucide-react";

export function App() {
  const auth = useAuth();

  // Full-screen spinner while we check the persisted auth state on startup.
  if (auth.loading && auth.auth === null) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-forge-600" />
      </div>
    );
  }

  return (
    <Routes>
      {auth.isAuthenticated ? (
        <>
          <Route path="/" element={<DashboardShell auth={auth} />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </>
      ) : (
        <>
          <Route path="/login" element={<LoginPage auth={auth} />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </>
      )}
    </Routes>
  );
}
