"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";
import {
  onAccessTokenRefreshed,
  onAuthCleared,
  onAuthRefreshed,
  getAccessToken,
  setAccessToken,
} from "@/lib/api/client";
import { clearStoredRefreshToken } from "@/lib/refresh-token";
import type { User } from "@/types";

interface AuthState {
  user: User | null;
  accessToken: string | null;
  hydrated: boolean;
  setAuth: (user: User, accessToken: string) => void;
  setAccessTokenOnly: (accessToken: string) => void;
  setUser: (user: User) => void;
  clearAuth: () => void;
  setHydrated: (v: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      hydrated: false,
      setAuth: (user, accessToken) => {
        setAccessToken(accessToken);
        set({ user, accessToken });
      },
      setAccessTokenOnly: (accessToken) => {
        setAccessToken(accessToken);
        set({ accessToken });
      },
      setUser: (user) => set({ user }),
      clearAuth: () => {
        setAccessToken(null);
        clearStoredRefreshToken();
        set({ user: null, accessToken: null });
      },
      setHydrated: (hydrated) => set({ hydrated }),
    }),
    {
      name: "sahiy-auth",
      // Faqat user saqlanadi — access token sessionStorage, refresh HttpOnly cookie.
      partialize: (s) => ({ user: s.user }),
      onRehydrateStorage: () => (state) => {
        const token = getAccessToken();
        if (state) state.accessToken = token;
        state?.setHydrated(true);
      },
    },
  ),
);

if (typeof window !== "undefined") {
  onAuthRefreshed((data) => {
    useAuthStore.getState().setAuth(data.user, data.access_token);
  });
  onAccessTokenRefreshed((token) => {
    useAuthStore.getState().setAccessTokenOnly(token);
  });
  onAuthCleared(() => {
    useAuthStore.getState().clearAuth();
  });
}
