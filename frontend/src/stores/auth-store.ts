import { create } from "zustand";

// Lightweight synchronous snapshot of auth state.
// The authoritative user data lives in AuthContext / TanStack Query.
// This store exists for code that needs a synchronous read (e.g. router loaders).

type AuthState = {
  isAuthenticated: boolean;
  setAuthenticated: (value: boolean) => void;
};

export const useAuthStore = create<AuthState>()((set) => ({
  isAuthenticated: false,
  setAuthenticated: (value) => set({ isAuthenticated: value }),
}));
