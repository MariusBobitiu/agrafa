import {
  focusManager,
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
export const ALERT_HISTORY_HEAD_REFETCH_INTERVAL_MS = 45_000;

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
  const seen = new Set<number>();
  const headWasFull = head.pagination.limit > 0 && head.alerts.length >= head.pagination.limit;
  const deduplicatedHead = {
    ...head,
    alerts: head.alerts.filter((alert) => {
      if (seen.has(alert.id)) return false;
      seen.add(alert.id);
      return true;
    }),
  };
  const replacement: InfiniteData<AlertPage, string | null> = {
    pages: [deduplicatedHead],
    pageParams: [null],
  };

  if (
    !current ||
    current.pages.length < 2 ||
    headWasFull ||
    deduplicatedHead.pagination.nextCursor == null
  ) {
    return replacement;
  }

  const firstPreservablePage = current.pageParams.findIndex(
    (pageParam, index) => index > 0 && pageParam === deduplicatedHead.pagination.nextCursor,
  );
  if (firstPreservablePage < 1) return replacement;

  let expectedPageParam: string | null = deduplicatedHead.pagination.nextCursor;
  for (let index = firstPreservablePage; index < current.pages.length; index += 1) {
    if (current.pageParams[index] !== expectedPageParam) break;

    const existingPage = current.pages[index];
    const alerts = existingPage.alerts.filter((alert) => {
      if (seen.has(alert.id)) return false;
      seen.add(alert.id);
      return true;
    });
    replacement.pages.push({ ...existingPage, alerts });
    replacement.pageParams.push(current.pageParams[index] ?? null);
    expectedPageParam = existingPage.pagination.nextCursor;
    if (expectedPageParam == null) break;
  }

  return replacement;
}

export async function reconcileAlertHistoryHead(
  queryClient: QueryClient,
  projectId: number,
  filters: AlertHistoryFilters,
  limit: number,
  shouldApply: () => boolean = () => true,
) {
  if (projectId <= 0) return;

  const historyKey = alertHistoryKeys.history(projectId, filters, limit);
  const head = await queryClient.fetchQuery({
    queryKey: alertHistoryKeys.historyHead(projectId, filters, limit),
    queryFn: ({ signal }) => alertsApi.listHistory(projectId, filters, limit, undefined, signal),
    staleTime: 0,
    retry: false,
  });
  if (!shouldApply()) return;

  await queryClient.cancelQueries({ queryKey: historyKey, exact: true }, { silent: true });
  if (!shouldApply()) return;

  queryClient.setQueryData<InfiniteData<AlertPage, string | null>>(historyKey, (current) =>
    replaceAlertHistoryHead(current, head),
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

  reconcileNow(projectId: number) {
    if (projectId <= 0) {
      this.reset();
      return Promise.resolve();
    }
    if (projectId !== this.projectId) this.reset(projectId);

    this.pendingGeneration += 1;
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

export function startAlertHistoryHeadReconciliation(
  sync: ActiveAlertHistorySync,
  projectId: number,
) {
  if (projectId <= 0) return () => {};

  const reconcile = () => void sync.reconcileNow(projectId);
  const interval = setInterval(reconcile, ALERT_HISTORY_HEAD_REFETCH_INTERVAL_MS);
  const unsubscribeFocus = focusManager.subscribe((isFocused) => {
    if (isFocused) reconcile();
  });

  return () => {
    clearInterval(interval);
    unsubscribeFocus();
  };
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
    const sync = new ActiveAlertHistorySync(reconcile);
    syncRef.current = sync;

    if (projectId <= 0) {
      sync.reset(projectId);
      return () => {
        shouldApply = false;
        if (syncRef.current === sync) syncRef.current = null;
      };
    }

    const stopReconciliation = startAlertHistoryHeadReconciliation(sync, projectId);
    return () => {
      shouldApply = false;
      stopReconciliation();
      sync.reset();
      if (syncRef.current === sync) syncRef.current = null;
    };
  }, [filters, limit, projectId, queryClient]);

  useEffect(() => {
    const sync = syncRef.current;
    if (sync == null) return;
    if (projectId <= 0 || activePage == null) {
      sync.reset(projectId);
      return;
    }
    void sync.update(projectId, activePage.alerts);
  }, [activeDataUpdatedAt, activePage, filters, limit, projectId]);
}

export function useAlertHistory(projectId: number, filters: AlertHistoryFilters, limit = 25) {
  return useInfiniteQuery({
    queryKey: alertHistoryKeys.history(projectId, filters, limit),
    queryFn: ({ pageParam, signal }) =>
      alertsApi.listHistory(projectId, filters, limit, pageParam ?? undefined, signal),
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
