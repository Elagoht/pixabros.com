import { create } from "zustand";
import { sessionService } from "@/services/session";

interface AuthState {
  user: ResponseMe | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  setUser: (user: ResponseMe | null) => void;
  setAuthenticated: (value: boolean) => void;
  checkAuth: () => Promise<void>;
  logout: () => Promise<void>;
}

let checkAuthPromise: Promise<void> | null = null;

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,

  setUser: (user) => set({ user }),
  setAuthenticated: (value) => set({ isAuthenticated: value }),

  checkAuth: async () => {
    if (checkAuthPromise) {
      return checkAuthPromise;
    }

    checkAuthPromise = (async () => {
      set({ isLoading: true });
      try {
        await sessionService.refresh();
        const currentUser = await sessionService.me();
        set({
          isAuthenticated: true,
          isLoading: false,
          user: currentUser,
        });
      } catch {
        set({ isAuthenticated: false, isLoading: false, user: null });
      }
      checkAuthPromise = null;
    })();

    return checkAuthPromise;
  },

  logout: async () => {
    try {
      await sessionService.delete();
    } finally {
      set({ isAuthenticated: false, user: null });
    }
    const isOnAuthPage =
      window.location.pathname.startsWith("/login") ||
      window.location.pathname.startsWith("/register");
    if (!isOnAuthPage) {
      const currentPath =
        window.location.pathname +
        window.location.search +
        window.location.hash;
      window.location.href = `/login?next=${encodeURIComponent(currentPath)}`;
    }
  },
}));
