import { create } from "zustand";
import { SessionService } from "@/services/session";
import { Navigation } from "@/utilities/navigation";

interface AuthState {
  user: ResponseWhoami | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  setUser: (user: ResponseWhoami | null) => void;
  setAuthenticated: (value: boolean) => void;
  setSession: (user: ResponseWhoami) => void;
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

  // Login already returns the admin, so the form can seed the store directly
  // instead of paying for a second round-trip to /whoami.
  setSession: (user) => set({ user, isAuthenticated: true, isLoading: false }),

  checkAuth: async () => {
    if (checkAuthPromise) {
      return checkAuthPromise;
    }

    checkAuthPromise = (async () => {
      set({ isLoading: true });
      try {
        const user = await SessionService.me();
        set({ isAuthenticated: true, isLoading: false, user });
      } catch {
        set({ isAuthenticated: false, isLoading: false, user: null });
      } finally {
        checkAuthPromise = null;
      }
    })();

    return checkAuthPromise;
  },

  logout: async () => {
    try {
      await SessionService.delete();
    } finally {
      set({ isAuthenticated: false, user: null });
    }
    Navigation.redirectToLogin();
  },
}));
