import {
  type InfiniteData,
  type QueryClient,
  queryOptions,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { alertsApi } from "@/data/alerts.ts";
import type {
  Alert,
  AlertHistoryFilters,
  AlertPage,
  AlertRuleCreateInput,
  AlertRuleUpdateInput,
} from "@/types/alert.ts";

export const ACTIVE_ALERTS_REFETCH_INTERVAL_MS = 12_000;

export const alertHistoryKeys = {
  active: (projectId: number) => ["alerts", projectId, "active"] as const,
  history: (projectId: number, filters: AlertHistoryFilters, limit: number) =>
    ["alerts", projectId, "history", filters, limit] as const,
  historyHead: (projectId: number, filters: AlertHistoryFilters, limit: number) =>
    ["alerts", projectId, "history-head", filters, limit] as const,
};

export function activeAlertsQueryOptions(projectId: number) {
  const enabled = projectId > 0;
  return queryOptions({
    queryKey: alertHistoryKeys.active(projectId),
    queryFn: () => alertsApi.listActive(projectId),
    enabled,
    refetchInterval: enabled ? ACTIVE_ALERTS_REFETCH_INTERVAL_MS : false,
    refetchOnWindowFocus: true,
  });
}

export function replaceAlertHistoryHead(
  current: InfiniteData<AlertPage, string | null> | undefined,
  head: AlertPage,
) {
  if (!current || current.pages.length === 0) return current;

  const seen = new Set<number>();
  const deduplicatedHead = {
    ...head,
    alerts: head.alerts.filter((alert) => {
      if (seen.has(alert.id)) return false;
      seen.add(alert.id);
      return true;
    }),
  };

  return {
    pages: [deduplicatedHead, ...current.pages.slice(1)],
    pageParams: current.pageParams,
  };
}

export async function reconcileAlertHistoryHead(
  queryClient: QueryClient,
  projectId: number,
  filters: AlertHistoryFilters,
  limit: number,
  shouldApply: () => boolean = () => true,
) {
  if (projectId <= 0) return;

  const head = await queryClient.fetchQuery({
    queryKey: alertHistoryKeys.historyHead(projectId, filters, limit),
    queryFn: () => alertsApi.listHistory(projectId, filters, limit),
    staleTime: 0,
    retry: false,
  });
  if (!shouldApply()) throw new Error("Alert history reconciliation superseded");

  queryClient.setQueryData<InfiniteData<AlertPage, string | null>>(
    alertHistoryKeys.history(projectId, filters, limit),
    (current) => replaceAlertHistoryHead(current, head),
  );
}

export function activeAlertIdentities(alerts: readonly Alert[]) {
  return alerts.map((alert) => `${alert.id}:${alert.status}`).sort();
}

export class ActiveAlertHistorySync {
  private projectId = 0;
  private previousIdentities: string[] | undefined;
  private pendingGeneration = 0;
  private completedGeneration = 0;
  private inFlight: Promise<void> | null = null;
  private reconcile: () => Promise<void>;

  constructor(reconcile: () => Promise<void>) {
    this.reconcile = reconcile;
  }

  setReconcile(reconcile: () => Promise<void>) {
    this.reconcile = reconcile;
  }

  reset(projectId = 0) {
    this.projectId = projectId;
    this.previousIdentities = undefined;
    this.pendingGeneration = 0;
    this.completedGeneration = 0;
  }

  update(projectId: number, alerts: readonly Alert[]) {
    if (projectId <= 0) {
      this.reset();
      return Promise.resolve();
    }

    const nextIdentities = activeAlertIdentities(alerts);
    if (projectId !== this.projectId || this.previousIdentities == null) {
      this.reset(projectId);
      this.previousIdentities = nextIdentities;
      return Promise.resolve();
    }

    const nextIdentitySet = new Set(nextIdentities);
    if (this.previousIdentities.some((identity) => !nextIdentitySet.has(identity))) {
      this.pendingGeneration += 1;
    }
    this.previousIdentities = nextIdentities;

    return this.drain(projectId);
  }

  private drain(projectId: number) {
    if (this.inFlight) return this.inFlight;
    if (this.pendingGeneration === this.completedGeneration) return Promise.resolve();

    const run = this.run(projectId);
    this.inFlight = run;
    void run.finally(() => {
      if (this.inFlight === run) this.inFlight = null;
    });
    return run;
  }

  private async run(projectId: number) {
    if (this.projectId !== projectId || this.completedGeneration >= this.pendingGeneration) return;

    const targetGeneration = this.pendingGeneration;
    try {
      await this.reconcile();
    } catch {
      return;
    }
    if (this.projectId !== projectId) return;
    this.completedGeneration = targetGeneration;
    await this.run(projectId);
  }
}

export function useAlerts(projectId: number) {
  return useQuery({
    queryKey: ["alerts", projectId],
    queryFn: () => alertsApi.listAlerts(projectId),
    enabled: projectId > 0,
  });
}

export function useActiveAlerts(projectId: number) {
  return useQuery(activeAlertsQueryOptions(projectId));
}

export function useAlertHistoryHeadSync(
  projectId: number,
  filters: AlertHistoryFilters,
  activePage: AlertPage | undefined,
  activeDataUpdatedAt: number,
  limit = 25,
) {
  const queryClient = useQueryClient();
  const syncRef = useRef<ActiveAlertHistorySync | null>(null);

  useEffect(() => {
    let shouldApply = true;
    const reconcile = () =>
      reconcileAlertHistoryHead(queryClient, projectId, filters, limit, () => shouldApply);
    const sync = syncRef.current ?? new ActiveAlertHistorySync(reconcile);
    syncRef.current = sync;
    sync.setReconcile(reconcile);

    if (projectId <= 0 || activePage == null) {
      sync.reset(projectId);
      return () => {
        shouldApply = false;
      };
    }
    void sync.update(projectId, activePage.alerts);
    return () => {
      shouldApply = false;
    };
  }, [activeDataUpdatedAt, activePage, filters, limit, projectId, queryClient]);
}

export function useAlertHistory(projectId: number, filters: AlertHistoryFilters, limit = 25) {
  return useInfiniteQuery({
    queryKey: alertHistoryKeys.history(projectId, filters, limit),
    queryFn: ({ pageParam }) =>
      alertsApi.listHistory(projectId, filters, limit, pageParam ?? undefined),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.pagination.nextCursor ?? undefined,
    enabled: projectId > 0,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
}

export function useAlertRules(projectId: number) {
  return useQuery({
    queryKey: ["alert-rules", projectId],
    queryFn: () => alertsApi.listRules(projectId),
    enabled: projectId > 0,
  });
}

export function useCreateAlertRule(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: AlertRuleCreateInput) => alertsApi.createRule(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
}

export function useUpdateAlertRule(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: AlertRuleUpdateInput }) =>
      alertsApi.updateRule(id, payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
}

export function useDeleteAlertRule(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => alertsApi.deleteRule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
}
