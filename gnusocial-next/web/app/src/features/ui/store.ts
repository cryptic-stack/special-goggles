import { create } from "zustand";
import type { DensityMode, UserRole } from "../../lib/types";

type Toast = {
  id: string;
  message: string;
};

type UiState = {
  density: DensityMode;
  reducedMotion: boolean;
  messagesEnabled: boolean;
  role: UserRole;
  commandOpen: boolean;
  toasts: Toast[];
  setDensity: (mode: DensityMode) => void;
  toggleReducedMotion: () => void;
  setMessagesEnabled: (enabled: boolean) => void;
  setRole: (role: UserRole) => void;
  setCommandOpen: (open: boolean) => void;
  pushToast: (message: string) => void;
  removeToast: (id: string) => void;
};

export const useUiStore = create<UiState>((set) => ({
  density: "default",
  reducedMotion: false,
  messagesEnabled: true,
  role: "admin",
  commandOpen: false,
  toasts: [],
  setDensity: (mode) => set({ density: mode }),
  toggleReducedMotion: () => set((state) => ({ reducedMotion: !state.reducedMotion })),
  setMessagesEnabled: (enabled) => set({ messagesEnabled: enabled }),
  setRole: (role) => set({ role }),
  setCommandOpen: (open) => set({ commandOpen: open }),
  pushToast: (message) =>
    set((state) => ({
      toasts: [...state.toasts, { id: `${Date.now()}-${Math.random()}`, message }]
    })),
  removeToast: (id) => set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) }))
}));
