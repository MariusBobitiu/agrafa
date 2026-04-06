import { create } from "zustand";
import { persist } from "zustand/middleware";

type UIState = {
  sidebarOpen: boolean;
  activeProjectId: number | null;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setActiveProjectId: (id: number | null) => void;
};

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      activeProjectId: null,
      toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setActiveProjectId: (id) => set({ activeProjectId: id }),
    }),
    {
      name: "agrafa-ui",
      partialize: (s) => ({
        sidebarOpen: s.sidebarOpen,
        activeProjectId: s.activeProjectId,
      }),
    },
  ),
);
