import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth.ts";

export function RequireGuest() {
  const { isLoading, isAuthenticated } = useAuth();

  if (isLoading) return null;

  if (isAuthenticated) {
    return <Navigate to="/overview" replace />;
  }

  return <Outlet />;
}
