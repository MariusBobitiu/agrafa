import {
  environmentManager,
  type InfiniteData,
  InfiniteQueryObserver,
  QueryClient,
  QueryObserver,
} from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ActiveAlertRow,
  AlertHistoryPagination,
  AlertHistoryTable,
  BackgroundRefreshError,
  EmptyState,
  ErrorState,
  TableSkeleton,
} from "@/app/alerts/alerts-page.tsx";
import { deduplicateAlerts, historyFilterTriggerClass } from "@/app/alerts/alert-presentation.ts";
import { alertsApi } from "@/data/alerts.ts";
import {
  ACTIVE_ALERTS_REFETCH_INTERVAL_MS,
  ActiveAlertHistorySync,
  activeAlertsQueryOptions,
  alertHistoryKeys,
  reconcileAlertHistoryHead,
} from "@/hooks/use-alerts.ts";
import { formatAlertDuration } from "@/lib/alert-duration.ts";
import type { Alert, AlertPage } from "@/types/alert.ts";

function alert(overrides: Partial<Alert> = {}): Alert {
  return {
    id: 1,
    project_id: 7,
    alert_rule_id: 11,
    rule_type: "node_offline",
    severity: "info",
    node_id: 21,
    node_name: "Worker one",
    node_identifier: "worker-one",
    service_id: null,
    service_name: null,
    title: "Node is offline and critically unreachable",
    message: "Connection refused",
    status: "active",
    triggered_at: "2026-08-23T10:00:00Z",
    resolved_at: null,
    ...overrides,
  };
}

function page(alerts: Alert[], nextCursor: string | null = null): AlertPage {
  return {
    alerts,
    pagination: { limit: 2, hasMore: nextCursor != null, nextCursor },
  };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
  environmentManager.setIsServer(() => true);
});

describe("alert presentation", () => {
  it("uses subdued dashed triggers only for unselected history filters", () => {
    expect(historyFilterTriggerClass(false)).toContain("border-input");
    expect(historyFilterTriggerClass(true)).toContain("border-input");
    expect(historyFilterTriggerClass(false)).toContain("border-dashed");
    expect(historyFilterTriggerClass(false)).toContain("opacity-90");
    expect(historyFilterTriggerClass(true)).toContain("border-solid");
    expect(historyFilterTriggerClass(true)).toContain("opacity-100");
  });

  it("uses authoritative severity and rule type instead of alert text", () => {
    const markup = renderToStaticMarkup(<ActiveAlertRow alert={alert()} />);

    expect(markup).toContain("Info");
    expect(markup).toContain("Node offline");
    expect(markup).not.toContain(">Critical<");
  });

  it("renders a resolved service unhealthy alert as an outage-like history row", () => {
    const outage = alert({
      rule_type: "service_unhealthy",
      severity: "critical",
      node_id: null,
      node_name: null,
      node_identifier: null,
      service_id: 42,
      service_name: "Public API",
      title: "Service health check failed",
      message: "HTTP 503",
      status: "resolved",
      resolved_at: "2026-08-23T11:12:00Z",
    });

    const markup = renderToStaticMarkup(<AlertHistoryTable alerts={[outage]} />);

    expect(markup).toContain("Service unhealthy");
    expect(markup).toContain("Public API");
    expect(markup).toContain("1h 12m");
    expect(markup).toContain("Critical");
    expect(markup).toContain(">resolved<");
  });

  it("formats resolved duration deterministically", () => {
    expect(formatAlertDuration("2026-08-23T10:00:00Z", "2026-08-23T10:00:42Z")).toBe("42s");
    expect(formatAlertDuration("2026-08-23T10:00:00Z", "2026-08-23T10:03:18Z")).toBe("3m 18s");
    expect(formatAlertDuration("2026-08-23T10:00:00Z", "2026-08-23T11:12:00Z")).toBe("1h 12m");
  });

  it("renders loading, empty, and retryable error states", () => {
    expect(renderToStaticMarkup(<TableSkeleton />)).toContain("animate-pulse");
    expect(renderToStaticMarkup(<EmptyState message="No resolved alerts." />)).toContain(
      "No resolved alerts.",
    );
    const errorMarkup = renderToStaticMarkup(
      <ErrorState message="Couldn’t load alert history." onRetry={() => {}} />,
    );
    expect(errorMarkup).toContain('role="alert"');
    expect(errorMarkup).toContain("Try again");
  });
});

describe("alert history pagination", () => {
  it("appends older pages without introducing duplicates", () => {
    const combined = deduplicateAlerts([
      alert({ id: 3 }),
      alert({ id: 2 }),
      alert({ id: 2 }),
      alert({ id: 1 }),
    ]);

    expect(combined.map((item) => item.id)).toEqual([3, 2, 1]);
  });

  it("preserves loaded rows after load-more failure and exposes retry", async () => {
    let nextPageAttempts = 0;
    const firstPage = page([alert({ id: 3 }), alert({ id: 2 })], "cursor-1");
    const secondPage = page([alert({ id: 2 }), alert({ id: 1 })]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new InfiniteQueryObserver(client, {
      queryKey: alertHistoryKeys.history(7, {}, 2),
      queryFn: ({ pageParam }) => {
        if (pageParam == null) return Promise.resolve(firstPage);
        nextPageAttempts += 1;
        return nextPageAttempts === 1
          ? Promise.reject(new Error("offline"))
          : Promise.resolve(secondPage);
      },
      initialPageParam: null as string | null,
      getNextPageParam: (lastPage) => lastPage.pagination.nextCursor ?? undefined,
    });
    const unsubscribe = observer.subscribe(() => {});
    await vi.waitFor(() => expect(observer.getCurrentResult().data?.pages).toHaveLength(1));

    await observer.fetchNextPage();

    const failed = observer.getCurrentResult();
    expect(failed.isFetchNextPageError).toBe(true);
    expect(failed.data?.pages).toEqual([firstPage]);
    const retryMarkup = renderToStaticMarkup(
      <AlertHistoryPagination
        hasNextPage={true}
        isFetchingNextPage={false}
        isFetchNextPageError={true}
        onLoadMore={() => {}}
      />,
    );
    expect(retryMarkup).toContain("loaded history is still shown");
    expect(retryMarkup).toContain("Try again");

    await observer.fetchNextPage();

    const loaded = observer.getCurrentResult().data?.pages.flatMap((item) => item.alerts) ?? [];
    expect(deduplicateAlerts(loaded).map((item) => item.id)).toEqual([3, 2, 1]);
    expect(nextPageAttempts).toBe(2);

    unsubscribe();
    client.clear();
  });
});

describe("alert live updates", () => {
  it("polls active alerts and shows a newly triggered alert without a manual refresh", async () => {
    environmentManager.setIsServer(() => false);
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T12:00:00Z");
    const triggered = alert({ id: 8 });
    const activeRequest = vi
      .spyOn(alertsApi, "listActive")
      .mockResolvedValueOnce(page([]))
      .mockResolvedValue(page([triggered]));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new QueryObserver(client, activeAlertsQueryOptions(7));
    const unsubscribe = observer.subscribe(() => {});

    await vi.advanceTimersByTimeAsync(0);
    expect(observer.getCurrentResult().data?.alerts).toEqual([]);

    await vi.advanceTimersByTimeAsync(ACTIVE_ALERTS_REFETCH_INTERVAL_MS);

    expect(activeRequest).toHaveBeenCalledTimes(2);
    expect(observer.getCurrentResult().data?.alerts).toEqual([triggered]);

    unsubscribe();
    client.clear();
  });

  it("removes a resolved alert and reconciles only the history head", async () => {
    environmentManager.setIsServer(() => false);
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T12:00:00Z");
    const active = alert({ id: 8 });
    const resolved = alert({
      id: 8,
      status: "resolved",
      resolved_at: "2026-08-23T12:00:10Z",
    });
    const oldPageZero = page([alert({ id: 5, status: "resolved" })], "cursor-0");
    const oldPageOne = page([alert({ id: 4, status: "resolved" })], "cursor-1");
    const oldPageTwo = page([alert({ id: 3, status: "resolved" })], "cursor-2");
    const pageParams = [null, "cursor-0", "cursor-1"];
    const cached: InfiniteData<AlertPage, string | null> = {
      pages: [oldPageZero, oldPageOne, oldPageTwo],
      pageParams,
    };
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(alertHistoryKeys.history(7, {}, 25), cached);
    vi.spyOn(alertsApi, "listActive")
      .mockResolvedValueOnce(page([active]))
      .mockResolvedValue(page([]));
    const historyRequest = vi
      .spyOn(alertsApi, "listHistory")
      .mockResolvedValue(page([resolved, alert({ id: 5, status: "resolved" })], "new-cursor"));
    const sync = new ActiveAlertHistorySync(() => reconcileAlertHistoryHead(client, 7, {}, 25));
    let activeDataUpdatedAt = 0;
    let latestSync = Promise.resolve();
    const observer = new QueryObserver(client, activeAlertsQueryOptions(7));
    const unsubscribe = observer.subscribe((result) => {
      if (result.data && result.dataUpdatedAt !== activeDataUpdatedAt) {
        activeDataUpdatedAt = result.dataUpdatedAt;
        latestSync = sync.update(7, result.data.alerts);
      }
    });

    await vi.advanceTimersByTimeAsync(0);
    await latestSync;
    expect(observer.getCurrentResult().data?.alerts).toEqual([active]);

    await vi.advanceTimersByTimeAsync(ACTIVE_ALERTS_REFETCH_INTERVAL_MS);
    await latestSync;

    const refreshed = client.getQueryData<InfiniteData<AlertPage, string | null>>(
      alertHistoryKeys.history(7, {}, 25),
    );
    expect(observer.getCurrentResult().data?.alerts).toEqual([]);
    expect(historyRequest).toHaveBeenCalledTimes(1);
    expect(historyRequest).toHaveBeenCalledWith(7, {}, 25);
    expect(refreshed?.pages[0]?.alerts.map((item) => item.id)).toEqual([8, 5]);
    expect(refreshed?.pages[1]).toBe(oldPageOne);
    expect(refreshed?.pages[2]).toBe(oldPageTwo);
    expect(refreshed?.pageParams).toBe(pageParams);

    unsubscribe();
    client.clear();
  });

  it("does not reconcile history when stable active identities are unchanged", async () => {
    const reconcile = vi.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const sync = new ActiveAlertHistorySync(reconcile);

    await sync.update(7, [alert({ id: 2 }), alert({ id: 1 })]);
    await sync.update(7, [alert({ id: 1 }), alert({ id: 2, title: "Updated title" })]);

    expect(reconcile).not.toHaveBeenCalled();
  });

  it("handles multiple simultaneous active changes with one bounded reconciliation", async () => {
    const reconcile = vi.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const sync = new ActiveAlertHistorySync(reconcile);

    await sync.update(7, [alert({ id: 1 }), alert({ id: 2 })]);
    await sync.update(7, [alert({ id: 2 }), alert({ id: 3 })]);
    await sync.update(7, [alert({ id: 3 }), alert({ id: 2 })]);

    expect(reconcile).toHaveBeenCalledTimes(1);
  });

  it("uses the filtered server head instead of inserting a mismatched resolved alert", async () => {
    const filters = { severity: "critical" } as const;
    const critical = alert({ id: 9, severity: "critical", status: "resolved" });
    const oldCritical = alert({ id: 6, severity: "critical", status: "resolved" });
    const warning = alert({ id: 8, severity: "warning", status: "resolved" });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData<InfiniteData<AlertPage, string | null>>(
      alertHistoryKeys.history(7, filters, 25),
      { pages: [page([oldCritical])], pageParams: [null] },
    );
    const historyRequest = vi
      .spyOn(alertsApi, "listHistory")
      .mockResolvedValue(page([critical, oldCritical]));

    await reconcileAlertHistoryHead(client, 7, filters, 25);

    const refreshed = client.getQueryData<InfiniteData<AlertPage, string | null>>(
      alertHistoryKeys.history(7, filters, 25),
    );
    expect(historyRequest).toHaveBeenCalledWith(7, filters, 25);
    expect(refreshed?.pages[0]?.alerts).toEqual([critical, oldCritical]);
    expect(refreshed?.pages[0]?.alerts).not.toContainEqual(warning);
    client.clear();
  });

  it("keeps stale active and history data visible through background failures", async () => {
    environmentManager.setIsServer(() => false);
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T12:00:00Z");
    const staleActive = page([alert({ id: 8 })]);
    vi.spyOn(alertsApi, "listActive")
      .mockResolvedValueOnce(staleActive)
      .mockRejectedValueOnce(new Error("offline"));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new QueryObserver(client, activeAlertsQueryOptions(7));
    const unsubscribe = observer.subscribe(() => {});
    await vi.advanceTimersByTimeAsync(0);

    await vi.advanceTimersByTimeAsync(ACTIVE_ALERTS_REFETCH_INTERVAL_MS);

    expect(observer.getCurrentResult().isError).toBe(true);
    expect(observer.getCurrentResult().data).toEqual(staleActive);
    expect(renderToStaticMarkup(<BackgroundRefreshError onRetry={() => {}} />)).toContain(
      "Showing saved active alerts",
    );

    const oldPageZero = page([alert({ id: 5, status: "resolved" })]);
    const oldPageOne = page([alert({ id: 4, status: "resolved" })]);
    const pageParams = [null, "cursor-0"];
    const cached: InfiniteData<AlertPage, string | null> = {
      pages: [oldPageZero, oldPageOne],
      pageParams,
    };
    client.setQueryData(alertHistoryKeys.history(7, {}, 25), cached);
    const newHead = page([alert({ id: 8, status: "resolved" })]);
    vi.spyOn(alertsApi, "listHistory")
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(newHead);
    const sync = new ActiveAlertHistorySync(() => reconcileAlertHistoryHead(client, 7, {}, 25));
    await sync.update(7, [alert({ id: 8 })]);
    await sync.update(7, []);

    expect(client.getQueryData(alertHistoryKeys.history(7, {}, 25))).toBe(cached);

    await Promise.resolve();
    await sync.update(7, []);

    const recovered = client.getQueryData<InfiniteData<AlertPage, string | null>>(
      alertHistoryKeys.history(7, {}, 25),
    );
    expect(recovered?.pages[0]).toEqual(newHead);
    expect(recovered?.pages[1]).toBe(oldPageOne);
    expect(recovered?.pageParams).toBe(pageParams);

    unsubscribe();
    client.clear();
  });

  it("resets active identity tracking on project switches", async () => {
    const reconcile = vi.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const sync = new ActiveAlertHistorySync(reconcile);

    await sync.update(7, [alert({ id: 8, project_id: 7 })]);
    await sync.update(9, []);

    expect(reconcile).not.toHaveBeenCalled();

    await sync.update(9, [alert({ id: 10, project_id: 9 })]);
    await sync.update(9, []);
    expect(reconcile).toHaveBeenCalledTimes(1);
  });

  it("does not run or schedule polling with an invalid project ID", async () => {
    environmentManager.setIsServer(() => false);
    vi.useFakeTimers();
    const activeRequest = vi.spyOn(alertsApi, "listActive").mockResolvedValue(page([]));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new QueryObserver(client, activeAlertsQueryOptions(0));
    const unsubscribe = observer.subscribe(() => {});

    await vi.advanceTimersByTimeAsync(ACTIVE_ALERTS_REFETCH_INTERVAL_MS * 5);

    expect(activeRequest).not.toHaveBeenCalled();
    expect(observer.getCurrentResult().fetchStatus).toBe("idle");

    unsubscribe();
    client.clear();
  });
});
