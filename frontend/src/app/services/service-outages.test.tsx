import {
  environmentManager,
  focusManager,
  type InfiniteData,
  InfiniteQueryObserver,
  QueryClient,
} from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  CurrentOutages,
  OutageAlertingNotConfigured,
  OutageHistoryPagination,
  ResolvedOutagesTable,
  ServiceOutagesContent,
} from "@/app/services/components/service-outages.tsx";
import {
  hasApplicableServiceOutageRule,
  otherServiceAlerts,
  resolvedServiceOutages,
  SERVICE_OUTAGE_HISTORY_LIMIT,
  serviceOutageAlerts,
  serviceOutageHistoryFilters,
} from "@/app/services/service-outage-utils.ts";
import { alertsApi } from "@/data/alerts.ts";
import {
  ALERT_HISTORY_HEAD_REFETCH_INTERVAL_MS,
  ActiveAlertHistorySync,
  alertHistoryKeys,
  reconcileAlertHistoryHead,
  startAlertHistoryHeadReconciliation,
} from "@/hooks/use-alerts.ts";
import type { Alert, AlertPage, AlertRule } from "@/types/alert.ts";
import type { ServiceAlert } from "@/types/service.ts";

function alert(overrides: Partial<Alert> = {}): Alert {
  return {
    id: 1,
    project_id: 7,
    alert_rule_id: 11,
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
    triggered_at: "2026-08-28T16:02:00Z",
    resolved_at: "2026-08-28T16:11:14Z",
    ...overrides,
  };
}

function serviceAlert(overrides: Partial<ServiceAlert> = {}): ServiceAlert {
  return {
    id: 1,
    rule_id: 11,
    rule_type: "service_unhealthy",
    severity: "critical",
    title: "Service health check failed",
    status: "active",
    triggered_at: "2026-08-28T11:45:30Z",
    ...overrides,
  };
}

function rule(overrides: Partial<AlertRule> = {}): AlertRule {
  return {
    id: 11,
    project_id: 7,
    node_id: null,
    service_id: null,
    rule_type: "service_unhealthy",
    threshold_value: null,
    severity: "critical",
    is_enabled: true,
    target_scope: "all",
    created_at: "2026-08-28T10:00:00Z",
    updated_at: "2026-08-28T10:00:00Z",
    ...overrides,
  };
}

function page(
  alerts: Alert[],
  nextCursor: string | null = null,
  limit = SERVICE_OUTAGE_HISTORY_LIMIT,
): AlertPage {
  return {
    alerts,
    pagination: { limit, hasMore: nextCursor != null, nextCursor },
  };
}

function renderWithRouter(node: React.ReactNode) {
  return renderToStaticMarkup(<MemoryRouter>{node}</MemoryRouter>);
}

function renderEmptyOutages(
  overrides: Partial<React.ComponentProps<typeof ServiceOutagesContent>>,
) {
  return renderWithRouter(
    <ServiceOutagesContent
      currentOutages={[]}
      resolvedOutages={[]}
      history={{ status: "success", nextPageStatus: "idle", hasNextPage: false }}
      ruleCoverage={{ status: "success", covered: true }}
      canManageRules={true}
      onLoadMore={() => {}}
      onRetryHistory={() => {}}
      onRetryRules={() => {}}
      now={Date.parse("2026-08-28T12:00:00Z")}
      {...overrides}
    />,
  );
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
  focusManager.setFocused(undefined);
  environmentManager.setIsServer(() => true);
});

describe("service outage presentation", () => {
  it("treats only service_unhealthy alert instances as service outages", () => {
    const currentOutage = serviceAlert({ id: 1 });
    const otherActiveAlert = serviceAlert({ id: 2, rule_type: "cpu_above_threshold" });
    expect(serviceOutageAlerts([currentOutage, otherActiveAlert])).toEqual([currentOutage]);
    expect(otherServiceAlerts([currentOutage, otherActiveAlert])).toEqual([otherActiveAlert]);

    const resolvedOutage = alert({ id: 3 });
    expect(
      resolvedServiceOutages(
        [
          resolvedOutage,
          alert({ id: 4, rule_type: "node_offline" }),
          alert({ id: 5, service_id: 99 }),
          alert({ id: 6, status: "active", resolved_at: null }),
        ],
        42,
      ),
    ).toEqual([resolvedOutage]);
  });

  it("renders an authoritative current outage with start, elapsed time, severity, and state", () => {
    const markup = renderToStaticMarkup(
      <CurrentOutages outages={[serviceAlert()]} now={Date.parse("2026-08-28T12:00:00Z")} />,
    );

    expect(markup).toContain("Current outage");
    expect(markup).toContain("14m 30s ago");
    expect(markup).toContain("Since");
    expect(markup).toContain("Critical");
    expect(markup).toContain("Ongoing");
  });

  it("renders resolved outage start, recovery, duration, and severity", () => {
    const markup = renderToStaticMarkup(<ResolvedOutagesTable outages={[alert()]} />);

    expect(markup).toContain("Started");
    expect(markup).toContain("Recovered");
    expect(markup).toContain("9m 14s");
    expect(markup).toContain("Critical");
  });

  it("renders the healthy no-outage empty state when alerting covers the service", () => {
    const markup = renderEmptyOutages({});

    expect(markup).toContain("No outages recorded");
    expect(markup).toContain("no recorded service-unhealthy alerts");
    expect(markup).not.toContain("isn’t configured");
  });

  it("distinguishes missing outage alerting from an empty covered history", () => {
    const markup = renderEmptyOutages({
      ruleCoverage: { status: "success", covered: false },
    });

    expect(markup).toContain("Outage alerting isn’t configured for this service");
    expect(markup).not.toContain("No outages recorded");
  });

  it("renders resolved outage history alongside missing alerting coverage", () => {
    const markup = renderEmptyOutages({
      resolvedOutages: [alert()],
      ruleCoverage: { status: "success", covered: false },
    });

    expect(markup).toContain('aria-label="Resolved outages"');
    expect(markup).toContain("Outage alerting isn’t configured for this service");
  });

  it("renders resolved outage history without a warning when alerting is covered", () => {
    const markup = renderEmptyOutages({ resolvedOutages: [alert()] });

    expect(markup).toContain('aria-label="Resolved outages"');
    expect(markup).not.toContain("isn’t configured");
  });

  it("keeps resolved outage history visible while coverage is loading", () => {
    const markup = renderEmptyOutages({
      resolvedOutages: [alert()],
      ruleCoverage: { status: "pending" },
    });

    expect(markup).toContain('aria-label="Resolved outages"');
    expect(markup).not.toContain("isn’t configured");
  });

  it("keeps resolved outage history visible when coverage fails to load", () => {
    const markup = renderEmptyOutages({
      resolvedOutages: [alert()],
      ruleCoverage: { status: "error" },
    });

    expect(markup).toContain('aria-label="Resolved outages"');
    expect(markup).not.toContain("isn’t configured");
  });

  it("keeps a current outage visible alongside missing alerting coverage", () => {
    const markup = renderEmptyOutages({
      currentOutages: [serviceAlert()],
      ruleCoverage: { status: "success", covered: false },
    });

    expect(markup).toContain('aria-label="Current outages"');
    expect(markup).toContain("Outage alerting isn’t configured for this service");
  });

  it("uses permission-aware alert-rule navigation", () => {
    const managerMarkup = renderWithRouter(<OutageAlertingNotConfigured canManageRules={true} />);
    const viewerMarkup = renderWithRouter(<OutageAlertingNotConfigured canManageRules={false} />);

    expect(managerMarkup).toContain("Configure rule");
    expect(viewerMarkup).toContain("View alert rules");
    expect(managerMarkup).toContain("/settings?tab=alert-rules");
  });
});

describe("service outage rule applicability", () => {
  it("counts enabled all-services and matching specific rules as coverage", () => {
    expect(hasApplicableServiceOutageRule([rule({ target_scope: "all" })], 42)).toBe(true);
    expect(
      hasApplicableServiceOutageRule([rule({ target_scope: "specific", service_id: 42 })], 42),
    ).toBe(true);
  });

  it("rejects another service, disabled rules, and other rule types", () => {
    expect(
      hasApplicableServiceOutageRule([rule({ target_scope: "specific", service_id: 99 })], 42),
    ).toBe(false);
    expect(hasApplicableServiceOutageRule([rule({ is_enabled: false })], 42)).toBe(false);
    expect(hasApplicableServiceOutageRule([rule({ rule_type: "node_offline" })], 42)).toBe(false);
  });
});

describe("service outage pagination", () => {
  it("requests resolved history with the service and service_unhealthy filters", async () => {
    const fetchRequest = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          alerts: [],
          pagination: { limit: 10, has_more: false, next_cursor: null },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    await alertsApi.listHistory(7, serviceOutageHistoryFilters(42), 10);

    const requestURL = fetchRequest.mock.calls[0]?.[0];
    expect(typeof requestURL).toBe("string");
    if (typeof requestURL !== "string") throw new Error("expected a string request URL");
    expect(requestURL).toContain("project_id=7");
    expect(requestURL).toContain("status=resolved");
    expect(requestURL).toContain("service_id=42");
    expect(requestURL).toContain("rule_type=service_unhealthy");
    expect(requestURL).toContain("limit=10");
  });

  it("loads older outages by cursor and retains existing rows after a load-more failure", async () => {
    const filters = serviceOutageHistoryFilters(42);
    const firstPage = page([alert({ id: 3 }), alert({ id: 2 })], "cursor-1");
    const secondPage = page([alert({ id: 1 })]);
    let olderAttempts = 0;
    const historyRequest = vi
      .spyOn(alertsApi, "listHistory")
      .mockImplementation((_projectId, _filters, _limit, before) => {
        if (before == null) return Promise.resolve(firstPage);
        olderAttempts += 1;
        return olderAttempts === 1
          ? Promise.reject(new Error("offline"))
          : Promise.resolve(secondPage);
      });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new InfiniteQueryObserver(client, {
      queryKey: alertHistoryKeys.history(7, filters, SERVICE_OUTAGE_HISTORY_LIMIT),
      queryFn: ({ pageParam, signal }) =>
        alertsApi.listHistory(
          7,
          filters,
          SERVICE_OUTAGE_HISTORY_LIMIT,
          pageParam ?? undefined,
          signal,
        ),
      initialPageParam: null as string | null,
      getNextPageParam: (lastPage) => lastPage.pagination.nextCursor ?? undefined,
    });
    const unsubscribe = observer.subscribe(() => {});
    await vi.waitFor(() => expect(observer.getCurrentResult().data?.pages).toEqual([firstPage]));

    await observer.fetchNextPage();

    expect(observer.getCurrentResult().isFetchNextPageError).toBe(true);
    expect(observer.getCurrentResult().data?.pages).toEqual([firstPage]);
    const failureMarkup = renderToStaticMarkup(
      <OutageHistoryPagination
        hasNextPage={true}
        isFetchingNextPage={false}
        isFetchNextPageError={true}
        onLoadMore={() => {}}
      />,
    );
    expect(failureMarkup).toContain("existing outages are still shown");
    expect(failureMarkup).toContain("Try again");

    await observer.fetchNextPage();

    expect(observer.getCurrentResult().data?.pages).toEqual([firstPage, secondPage]);
    expect(historyRequest.mock.calls.at(-1)?.[3]).toBe("cursor-1");
    expect(
      renderToStaticMarkup(
        <OutageHistoryPagination
          hasNextPage={true}
          isFetchingNextPage={false}
          isFetchNextPageError={false}
          onLoadMore={() => {}}
        />,
      ),
    ).toContain("Load older outages");

    unsubscribe();
    client.clear();
  });

  it("keeps service caches isolated when the route switches services", async () => {
    const filters42 = serviceOutageHistoryFilters(42);
    const filters43 = serviceOutageHistoryFilters(43);
    const key42 = alertHistoryKeys.history(7, filters42, SERVICE_OUTAGE_HISTORY_LIMIT);
    const key43 = alertHistoryKeys.history(7, filters43, SERVICE_OUTAGE_HISTORY_LIMIT);
    const service42Data: InfiniteData<AlertPage, string | null> = {
      pages: [page([alert({ id: 42, service_id: 42 })])],
      pageParams: [null],
    };
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(key42, service42Data);
    vi.spyOn(alertsApi, "listHistory").mockResolvedValue(page([alert({ id: 43, service_id: 43 })]));

    await reconcileAlertHistoryHead(client, 7, filters43, SERVICE_OUTAGE_HISTORY_LIMIT);

    expect(key43).not.toEqual(key42);
    expect(client.getQueryData(key42)).toBe(service42Data);
    expect(
      client
        .getQueryData<InfiniteData<AlertPage, string | null>>(key43)
        ?.pages[0]?.alerts.map((item) => item.service_id),
    ).toEqual([43]);
    client.clear();
  });
});

describe("service outage live reconciliation", () => {
  it("moves a recovered current outage into the filtered resolved history", async () => {
    const filters = serviceOutageHistoryFilters(42);
    const historyKey = alertHistoryKeys.history(7, filters, SERVICE_OUTAGE_HISTORY_LIMIT);
    const recovered = alert({ id: 8 });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData<InfiniteData<AlertPage, string | null>>(historyKey, {
      pages: [page([])],
      pageParams: [null],
    });
    const historyRequest = vi.spyOn(alertsApi, "listHistory").mockResolvedValue(page([recovered]));
    const sync = new ActiveAlertHistorySync(() =>
      reconcileAlertHistoryHead(client, 7, filters, SERVICE_OUTAGE_HISTORY_LIMIT),
    );

    await sync.update(7, [{ id: 8, status: "active" }]);
    await sync.update(7, []);

    expect(historyRequest).toHaveBeenCalledTimes(1);
    expect(historyRequest.mock.calls[0]?.slice(0, 4)).toEqual([
      7,
      filters,
      SERVICE_OUTAGE_HISTORY_LIMIT,
      undefined,
    ]);
    expect(
      client
        .getQueryData<InfiniteData<AlertPage, string | null>>(historyKey)
        ?.pages[0]?.alerts.map((item) => item.id),
    ).toEqual([8]);
    client.clear();
  });

  it("finds a short-lived outage through bounded head reconciliation", async () => {
    vi.useFakeTimers();
    const filters = serviceOutageHistoryFilters(42);
    const historyKey = alertHistoryKeys.history(7, filters, SERVICE_OUTAGE_HISTORY_LIMIT);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData<InfiniteData<AlertPage, string | null>>(historyKey, {
      pages: [page([])],
      pageParams: [null],
    });
    const historyRequest = vi
      .spyOn(alertsApi, "listHistory")
      .mockResolvedValue(page([alert({ id: 9 })]));
    const sync = new ActiveAlertHistorySync(() =>
      reconcileAlertHistoryHead(client, 7, filters, SERVICE_OUTAGE_HISTORY_LIMIT),
    );
    await sync.update(7, []);
    const stop = startAlertHistoryHeadReconciliation(sync, 7);

    await vi.advanceTimersByTimeAsync(ALERT_HISTORY_HEAD_REFETCH_INTERVAL_MS);

    expect(historyRequest).toHaveBeenCalledTimes(1);
    expect(historyRequest.mock.calls[0]?.[3]).toBeUndefined();
    expect(
      client
        .getQueryData<InfiniteData<AlertPage, string | null>>(historyKey)
        ?.pages[0]?.alerts.map((item) => item.id),
    ).toEqual([9]);

    stop();
    client.clear();
  });

  it("reconciles only the head and preserves already-loaded older pages", async () => {
    const filters = serviceOutageHistoryFilters(42);
    const historyKey = alertHistoryKeys.history(7, filters, SERVICE_OUTAGE_HISTORY_LIMIT);
    const head = page(
      Array.from({ length: SERVICE_OUTAGE_HISTORY_LIMIT }, (_, index) => alert({ id: 20 - index })),
      "cursor-older",
    );
    const olderPage = page([alert({ id: 10 }), alert({ id: 9 })]);
    const pageParams = [null, "cursor-older"];
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData<InfiniteData<AlertPage, string | null>>(historyKey, {
      pages: [head, olderPage],
      pageParams,
    });
    const historyRequest = vi.spyOn(alertsApi, "listHistory").mockResolvedValue({
      ...head,
      alerts: [...head.alerts],
    });

    await reconcileAlertHistoryHead(client, 7, filters, SERVICE_OUTAGE_HISTORY_LIMIT);

    const reconciled = client.getQueryData<InfiniteData<AlertPage, string | null>>(historyKey);
    expect(historyRequest).toHaveBeenCalledTimes(1);
    expect(historyRequest.mock.calls[0]?.[3]).toBeUndefined();
    expect(reconciled?.pages[1]).toBe(olderPage);
    expect(reconciled?.pageParams).toEqual(pageParams);
    client.clear();
  });
});
