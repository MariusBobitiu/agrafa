import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createContext, useCallback, useEffect, useMemo, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { authApi } from "@/data/auth.ts";
import { clearAuthRedirect, resolveAuthRedirect } from "@/lib/auth-redirect.ts";
import { ApiError } from "@/lib/fetch-client.ts";
import { useAuthStore } from "@/stores/auth-store.ts";
import type { LoginInput, RegisterInput, User } from "@/types/auth.ts";

type AuthNavigationOptions = {
  redirectTo?: string | null;
};

type AuthContextValue = {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (input: LoginInput, options?: AuthNavigationOptions) => Promise<void>;
  register: (input: RegisterInput, options?: AuthNavigationOptions) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<User | null>;
};

export const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setAuthenticated = useAuthStore((s) => s.setAuthenticated);

  const fetchCurrentUser = useCallback(async () => {
    try {
      return await authApi.me();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) return null;
      throw err;
    }
  }, []);

  const { data, isLoading } = useQuery({
    queryKey: ["auth", "me"],
    queryFn: fetchCurrentUser,
    staleTime: Infinity,
    retry: false,
  });

  const user = data?.user ?? null;
  const isAuthenticated = user !== null;

  useEffect(() => {
    setAuthenticated(isAuthenticated);
  }, [isAuthenticated, setAuthenticated]);

  const refreshUser = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    const next = await queryClient.fetchQuery({
      queryKey: ["auth", "me"],
      queryFn: fetchCurrentUser,
      staleTime: 0,
      retry: false,
    });
    return next?.user ?? null;
  }, [fetchCurrentUser, queryClient]);

  const login = useCallback(
    async (input: LoginInput, options?: AuthNavigationOptions) => {
      await authApi.login(input);
      const nextUser = await refreshUser();
      const redirectTo = resolveAuthRedirect({
        candidate: options?.redirectTo,
        fallback: "/overview",
      });

      if (nextUser && !nextUser.email_verified && !redirectTo.startsWith("/invite")) {
        navigate("/verify-email", { replace: true });
        return;
      }

      navigate(redirectTo, { replace: true });
    },
    [navigate, refreshUser],
  );

  const register = useCallback(
    async (input: RegisterInput, options?: AuthNavigationOptions) => {
      await authApi.register(input);
      const nextUser = await refreshUser();
      const redirectTo = resolveAuthRedirect({
        candidate: options?.redirectTo,
        fallback: "/onboarding",
      });

      if (nextUser && !nextUser.email_verified && !redirectTo.startsWith("/invite")) {
        await authApi.sendVerifyEmail();
        navigate("/verify-email", { replace: true });
        return;
      }

      navigate(redirectTo, { replace: true });
    },
    [navigate, refreshUser],
  );

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      clearAuthRedirect();
      queryClient.clear();
      setAuthenticated(false);
      navigate("/sign-in", { replace: true });
    }
  }, [queryClient, setAuthenticated, navigate]);

  const value = useMemo<AuthContextValue>(
    () => ({ user, isLoading, isAuthenticated, login, register, logout, refreshUser }),
    [user, isLoading, isAuthenticated, login, register, logout, refreshUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
