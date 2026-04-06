import { create } from "zustand";
import type { CheckType, ServiceExecutionMode } from "@/types/service.ts";

export type ServiceCreationDraft = {
  name: string;
  executionMode: ServiceExecutionMode;
  nodeId: string;
  checkType: CheckType;
  checkTarget: string;
};

type ServiceCreationState = {
  draft: ServiceCreationDraft | null;
  pendingNodeId: string | null;
  resumeRequested: boolean;
  saveDraft: (draft: ServiceCreationDraft) => void;
  markResume: (nodeId?: string | null) => void;
  consumeResume: () => void;
  clear: () => void;
};

export const useServiceCreationStore = create<ServiceCreationState>((set) => ({
  draft: null,
  pendingNodeId: null,
  resumeRequested: false,
  saveDraft: (draft) => set({ draft, pendingNodeId: null, resumeRequested: false }),
  markResume: (nodeId) =>
    set((state) => ({
      draft: state.draft,
      pendingNodeId: nodeId ?? null,
      resumeRequested: true,
    })),
  consumeResume: () => set({ resumeRequested: false }),
  clear: () => set({ draft: null, pendingNodeId: null, resumeRequested: false }),
}));
