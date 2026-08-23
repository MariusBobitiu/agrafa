import {
  environmentManager,
  InfiniteQueryObserver,
  QueryClient,
  QueryObserver,
  type InfiniteData,
} from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { servicesApi, toHistoryObservation } from "@/data/services.ts";
import type {
  CheckType,
  Service,
  ServiceHistoryObservation,
  ServiceHistoryPage,
} from "@/types/service.ts";
import type { ServiceHistoryWindow } from "@/types/service.ts";
import {
  applyServiceDetailStreamPayload,
  serviceHistoryRangeRefetchInterval,
  serviceHistoryKeys,
  serviceHistoryWindowQueryOptions,
} from "@/hooks/use-services.ts";
import {
  ObservationTooltip,
  RecentChecksHeading,
  RecentChecksList,
  RecentChecksPagination,
  ServiceHealthLoadingState,
  ServiceHistoryEmptyState,
  ServiceHistoryRefreshError,
  ServiceHistoryRow,
  ServiceHistorySummaryMetrics,
  ServiceHistorySuccessCount,
} from "./components/service-history.tsx";
import {
  buildHistoryChartData,
  calculateHistoryMetrics,
  deduplicateHistory,
  formatLatency,
  formatObservationTooltipTime,
  getHistoryRowPresentation,
} from "./service-history-utils.ts";

function observation(
  id: number,
  overrides: Partial<ServiceHistoryObservation> = {},
): ServiceHistoryObservation {
  return {
    id,
    serviceId: 7,
    nodeId: 2,
    checkType: "http" as CheckType,
    source: "managed",
    observedAt: new Date(Date.UTC(2026, 7, 23, 12, 0, id)).toISOString().replace(".000Z", "Z"),
    isSuccess: true,
    statusCode: 200,
    latencyMs: 20,
    message: "200 OK",
    metadata: {},
    ...overrides,
  };
}

function historyPage(ids: number[], nextCursor: string | null = null): ServiceHistoryPage {
  return {
    observations: ids.map((id) => observation(id)),
    pagination: {
      limit: 20,
      hasMore: nextCursor != null,
      nextCursor,
    },
  };
}

function historyWindow(
  observations: ServiceHistoryObservation[],
  overrides: Partial<ServiceHistoryWindow> = {},
): ServiceHistoryWindow {
  const successfulChecks = observations.filter((item) => item.isSuccess).length;
  return {
    observations,
    isTruncated: false,
    from: "2026-08-16T00:00:00Z",
    to: "2026-08-24T00:00:00Z",
    summary: {
      from: "2026-08-16T00:00:00Z",
      to: "2026-08-24T00:00:00Z",
      totalChecks: observations.length,
      successfulChecks,
      uptimePercent:
        observations.length > 0 ? (successfulChecks / observations.length) * 100 : null,
      averageLatencyMs: observations[0]?.latencyMs ?? null,
      lastCheckedAt: observations[0]?.observedAt ?? null,
    },
    ...overrides,
  };
}

function serviceSnapshot(): Service {
  return {
    id: 7,
    project_id: 1,
    node_id: 2,
    execution_mode: "managed",
    name: "API",
    check_type: "http",
    check_target: "https://example.com/health",
    status: "healthy",
    last_checked_at: null,
    consecutive_failures: 0,
    active_alert_count: 0,
    latest_health_check: null,
    created_at: "2026-08-23T10:00:00Z",
    updated_at: "2026-08-23T10:00:00Z",
  };
}

function historyEntry(item: ServiceHistoryObservation) {
  return {
    id: item.id,
    service_id: item.serviceId,
    node_id: item.nodeId,
    check_type: item.checkType,
    source: item.source,
    observed_at: item.observedAt,
    is_success: item.isSuccess,
    status_code: item.statusCode,
    response_time_ms: item.latencyMs,
    message: item.message,
    metadata: item.metadata,
  };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
  environmentManager.setIsServer(() => true);
});

describe("service history authoritative bounds", () => {
  it("uses clamped summary bounds for observations and cache metadata", async () => {
    const serverTo = new Date("2026-08-23T12:00:00Z");
    const serverFrom = new Date(serverTo.getTime() - 24 * 60 * 60 * 1_000);
    const requestedTo = new Date(serverTo.getTime() + 30_000);
    const requestedFrom = new Date(requestedTo.getTime() - 24 * 60 * 60 * 1_000);
    vi.spyOn(servicesApi, "history").mockResolvedValue({
      ...historyPage([]),
      observations: [
        observation(1, { observedAt: new Date(serverTo.getTime() + 15_000).toISOString() }),
      ],
    });
    vi.spyOn(servicesApi, "historySummary").mockResolvedValue({
      from: serverFrom.toISOString(),
      to: serverTo.toISOString(),
      totalChecks: 0,
      successfulChecks: 0,
      uptimePercent: null,
      averageLatencyMs: null,
      lastCheckedAt: null,
    });

    const result = await servicesApi.historyWindow(7, requestedFrom, requestedTo);

    expect(result.from).toBe(serverFrom.toISOString());
    expect(result.to).toBe(serverTo.toISOString());
    expect(result.observations).toEqual([]);
  });
});

describe("service history loading states", () => {
  it("is pending without data on first load and retains the previous range while refetching", async () => {
    const firstWindow = historyWindow([observation(1)]);
    const secondWindow = historyWindow([observation(2)]);
    let resolveSecond: ((value: ServiceHistoryWindow) => void) | undefined;
    let requestCount = 0;

    vi.spyOn(servicesApi, "historyWindow").mockImplementation(() => {
      requestCount += 1;
      if (requestCount === 1) return Promise.resolve(firstWindow);
      return new Promise((resolve) => {
        resolveSecond = resolve;
      });
    });

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new QueryObserver(client, serviceHistoryWindowQueryOptions(7, 1));
    const unsubscribe = observer.subscribe(() => {});

    expect(observer.getCurrentResult().isPending).toBe(true);
    expect(observer.getCurrentResult().data).toBeUndefined();

    await vi.waitFor(() => expect(observer.getCurrentResult().data).toEqual(firstWindow));
    observer.setOptions(serviceHistoryWindowQueryOptions(7, 6));

    const refetching = observer.getCurrentResult();
    expect(refetching.isPending).toBe(false);
    expect(refetching.isFetching).toBe(true);
    expect(refetching.isPlaceholderData).toBe(true);
    expect(refetching.data).toEqual(firstWindow);

    resolveSecond?.(secondWindow);
    await vi.waitFor(() => expect(observer.getCurrentResult().data).toEqual(secondWindow));

    unsubscribe();
    client.clear();
  });

  it("uses a service-health-shaped skeleton for the initial range load", () => {
    const markup = renderToStaticMarkup(<ServiceHealthLoadingState />);

    expect(markup).toContain('aria-label="Loading service health"');
    expect(markup).toContain("grid-cols-3");
    expect(markup).toContain("h-56");
  });

  it("retains range data and offers retry after a background refresh error", async () => {
    const staleWindow = historyWindow([observation(1)]);
    vi.spyOn(servicesApi, "historyWindow")
      .mockResolvedValueOnce(staleWindow)
      .mockRejectedValueOnce(new Error("offline"));

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new QueryObserver(client, serviceHistoryWindowQueryOptions(7, 24));
    const unsubscribe = observer.subscribe(() => {});

    await vi.waitFor(() => expect(observer.getCurrentResult().data).toBe(staleWindow));
    await observer.refetch();

    const failedRefresh = observer.getCurrentResult();
    expect(failedRefresh.isError).toBe(true);
    expect(failedRefresh.data).toBe(staleWindow);
    expect(failedRefresh.data?.observations).toHaveLength(1);

    const markup = renderToStaticMarkup(<ServiceHistoryRefreshError onRetry={() => {}} />);
    expect(markup).toContain("Showing saved data");
    expect(markup).toContain("Try again");

    unsubscribe();
    client.clear();
  });

  it("keeps an empty stale range visible with its refresh retry", async () => {
    const emptyWindow = historyWindow([]);
    vi.spyOn(servicesApi, "historyWindow")
      .mockResolvedValueOnce(emptyWindow)
      .mockRejectedValueOnce(new Error("offline"));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new QueryObserver(client, serviceHistoryWindowQueryOptions(7, 24));
    const unsubscribe = observer.subscribe(() => {});
    await vi.waitFor(() => expect(observer.getCurrentResult().data).toBe(emptyWindow));
    await observer.refetch();

    const failedRefresh = observer.getCurrentResult();
    expect(failedRefresh.isError).toBe(true);
    expect(failedRefresh.data?.observations).toEqual([]);
    const markup = renderToStaticMarkup(
      <ServiceHistoryEmptyState refreshError={failedRefresh.isError} onRetry={() => {}} />,
    );

    expect(markup).toContain("No check history in this range.");
    expect(markup).toContain("History refresh failed. Showing saved data.");
    expect(markup).toContain("Try again");

    unsubscribe();
    client.clear();
  });
});

describe("service history SSE refresh", () => {
  it("grows page zero without creating a cursor gap or changing older pages", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const key = serviceHistoryKeys.list(7, 20);
    const pageZeroIDs = Array.from({ length: 20 }, (_, index) => 20 - index);
    const olderPage = historyPage([99, 98], "cursor-2");
    const cached: InfiniteData<ServiceHistoryPage, string | null> = {
      pages: [historyPage(pageZeroIDs, "cursor-1"), olderPage],
      pageParams: [null, "cursor-1"],
    };
    const inactiveKey = serviceHistoryKeys.list(7, 50);
    const inactiveCached: InfiniteData<ServiceHistoryPage, string | null> = {
      pages: [historyPage([60, 59], "cursor-1")],
      pageParams: [null],
    };
    client.setQueryData(key, cached);
    client.setQueryData(inactiveKey, inactiveCached);
    const observer = new QueryObserver(client, {
      queryKey: key,
      queryFn: () => Promise.resolve(cached),
      staleTime: Infinity,
    });
    const unsubscribe = observer.subscribe(() => {});
    const historyRequest = vi.spyOn(servicesApi, "history");
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const streamed = observation(21, { latencyMs: null });

    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(streamed),
    });

    const refreshed = client.getQueryData<InfiniteData<ServiceHistoryPage, string | null>>(key);
    expect(historyRequest).not.toHaveBeenCalled();
    expect(invalidate).not.toHaveBeenCalled();
    expect(refreshed?.pages[0]?.observations.map((item) => item.id)).toEqual([21, ...pageZeroIDs]);
    expect(refreshed?.pages[0]?.observations[0]?.latencyMs).toBeNull();
    expect(refreshed?.pages[0]?.pagination.nextCursor).toBe("cursor-1");
    expect(refreshed?.pages[1]).toBe(olderPage);
    expect(refreshed?.pageParams).toBe(cached.pageParams);
    expect(refreshed?.pages.flatMap((page) => page.observations).map((item) => item.id)).toEqual([
      21,
      ...pageZeroIDs,
      99,
      98,
    ]);
    expect(client.getQueryData(inactiveKey)).toBe(inactiveCached);
    expect(client.getQueryData(["services", "detail", 7])).toEqual({
      service: serviceSnapshot(),
    });

    unsubscribe();
    client.clear();
  });

  it("is idempotent by ID and retains newest-first ordering", async () => {
    const client = new QueryClient();
    const key = serviceHistoryKeys.list(7, 20);
    const cached: InfiniteData<ServiceHistoryPage, string | null> = {
      pages: [historyPage([3, 1])],
      pageParams: [null],
    };
    client.setQueryData(key, cached);
    const observer = new QueryObserver(client, {
      queryKey: key,
      queryFn: () => Promise.resolve(cached),
      staleTime: Infinity,
    });
    const unsubscribe = observer.subscribe(() => {});
    const repeated = observation(2);

    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(repeated),
    });
    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(repeated),
    });

    const result = client.getQueryData<InfiniteData<ServiceHistoryPage, string | null>>(key);
    expect(result?.pages[0]?.observations.map((item) => item.id)).toEqual([3, 2, 1]);

    unsubscribe();
    client.clear();
  });

  it("updates only loaded in-range windows and does not create unrelated queries", async () => {
    const client = new QueryClient();
    const rangeKey = serviceHistoryKeys.window(7, 24);
    const streamedAt = new Date();
    const rangeData = historyWindow(
      [observation(1, { observedAt: new Date(streamedAt.getTime() - 60_000).toISOString() })],
      {
        from: new Date(streamedAt.getTime() - 24 * 60 * 60 * 1_000).toISOString(),
        to: new Date(streamedAt.getTime() + 60_000).toISOString(),
      },
    );
    rangeData.summary = { ...rangeData.summary, totalChecks: 4_000, successfulChecks: 3_999 };
    client.setQueryData(rangeKey, rangeData);
    const streamed = observation(62, { observedAt: streamedAt.toISOString() });

    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(
        observation(62, {
          checkType: "tcp",
          observedAt: streamed.observedAt,
          statusCode: null,
          latencyMs: null,
          message: "connected",
        }),
      ),
    });

    const updated = client.getQueryData<ServiceHistoryWindow>(rangeKey);
    expect(updated?.observations[0]).toMatchObject({ id: 62, checkType: "tcp", latencyMs: null });
    expect(updated?.summary).toEqual(rangeData.summary);
    expect(client.getQueryData(serviceHistoryKeys.window(7, 6))).toBeUndefined();
    expect(client.getQueryData(serviceHistoryKeys.list(8, 20))).toBeUndefined();
    client.clear();
  });

  it("does not add a future-dated observation to a bounded range cache", async () => {
    const client = new QueryClient();
    const rangeKey = serviceHistoryKeys.window(7, 24);
    const range = historyWindow([observation(1)]);
    client.setQueryData(rangeKey, range);

    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(
        observation(99, { observedAt: "2026-08-25T00:00:00Z", latencyMs: null }),
      ),
    });

    expect(client.getQueryData<ServiceHistoryWindow>(rangeKey)).toEqual(range);
    client.clear();
  });

  it("cancels older in-flight recent and range responses before applying SSE data", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const recentKey = serviceHistoryKeys.list(7, 20);
    const rangeKey = serviceHistoryKeys.window(7, 24);
    const initialPage = historyPage([2, 1]);
    const initialRange = historyWindow([observation(2), observation(1)]);
    let resolveRecent: ((page: ServiceHistoryPage) => void) | undefined;
    let resolveRange: ((window: ServiceHistoryWindow) => void) | undefined;
    const recentObserver = new InfiniteQueryObserver(client, {
      queryKey: recentKey,
      queryFn: () =>
        new Promise<ServiceHistoryPage>((resolve) => {
          resolveRecent = resolve;
        }),
      initialPageParam: null as string | null,
      getNextPageParam: (lastPage) => lastPage.pagination.nextCursor ?? undefined,
      initialData: { pages: [initialPage], pageParams: [null] },
      staleTime: Infinity,
    });
    const rangeObserver = new QueryObserver(client, {
      queryKey: rangeKey,
      queryFn: () =>
        new Promise<ServiceHistoryWindow>((resolve) => {
          resolveRange = resolve;
        }),
      initialData: initialRange,
      staleTime: Infinity,
    });
    const unsubscribeRecent = recentObserver.subscribe(() => {});
    const unsubscribeRange = rangeObserver.subscribe(() => {});
    const recentFetch = recentObserver.refetch();
    const rangeFetch = rangeObserver.refetch();
    await vi.waitFor(() => {
      expect(recentObserver.getCurrentResult().isFetching).toBe(true);
      expect(rangeObserver.getCurrentResult().isFetching).toBe(true);
    });

    const streamed = observation(3);
    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(streamed),
    });
    resolveRecent?.(initialPage);
    resolveRange?.(initialRange);
    await Promise.all([recentFetch, rangeFetch]);

    expect(
      client.getQueryData<InfiniteData<ServiceHistoryPage, string | null>>(recentKey)?.pages[0]
        ?.observations[0]?.id,
    ).toBe(3);
    expect(client.getQueryData<ServiceHistoryWindow>(rangeKey)?.observations[0]?.id).toBe(3);

    unsubscribeRecent();
    unsubscribeRange();
    client.clear();
  });

  it("keeps the authoritative range refresh due at its original cadence", async () => {
    environmentManager.setIsServer(() => false);
    vi.useFakeTimers();
    const startedAt = new Date("2026-08-23T12:00:00Z");
    vi.setSystemTime(startedAt);
    const client = new QueryClient();
    const rangeKey = serviceHistoryKeys.window(7, 24);
    client.setQueryData<ServiceHistoryWindow>(
      rangeKey,
      historyWindow([], {
        from: new Date(startedAt.getTime() - 24 * 60 * 60 * 1_000).toISOString(),
        to: new Date(startedAt.getTime() + 60_000).toISOString(),
      }),
    );
    const authoritativeFetch = vi
      .spyOn(servicesApi, "historyWindow")
      .mockResolvedValue(historyWindow([observation(100)]));
    const observer = new QueryObserver(client, {
      ...serviceHistoryWindowQueryOptions(7, 24),
      staleTime: Infinity,
    });
    const unsubscribe = observer.subscribe(() => {});

    for (const seconds of [15, 30, 45]) {
      await vi.advanceTimersByTimeAsync(15_000);
      const streamed = observation(seconds, { observedAt: new Date().toISOString() });
      await applyServiceDetailStreamPayload(client, 7, {
        service: serviceSnapshot(),
        observation: historyEntry(streamed),
      });
    }

    expect(authoritativeFetch).not.toHaveBeenCalled();
    expect(serviceHistoryRangeRefetchInterval(client.getQueryState(rangeKey)!)).toBe(15_000);
    await vi.advanceTimersByTimeAsync(14_999);
    expect(authoritativeFetch).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(authoritativeFetch).toHaveBeenCalledTimes(1);

    unsubscribe();
    client.clear();
  });

  it("allows a full range cache to grow without falsely changing truncation metadata", async () => {
    const client = new QueryClient();
    const rangeKey = serviceHistoryKeys.window(7, 168);
    const cached = Array.from({ length: 2_000 }, (_, index) =>
      observation(index + 1, { observedAt: new Date(Date.now() - index * 1_000).toISOString() }),
    );
    client.setQueryData<ServiceHistoryWindow>(
      rangeKey,
      historyWindow(cached, {
        from: new Date(Date.now() - 168 * 60 * 60 * 1_000).toISOString(),
        to: new Date(Date.now() + 60_000).toISOString(),
      }),
    );
    const streamed = observation(2_001, { observedAt: new Date().toISOString() });

    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(streamed),
    });
    await applyServiceDetailStreamPayload(client, 7, {
      service: serviceSnapshot(),
      observation: historyEntry(streamed),
    });

    const updated = client.getQueryData<ServiceHistoryWindow>(rangeKey);
    expect(updated?.observations).toHaveLength(2_001);
    expect(updated?.isTruncated).toBe(false);
    client.clear();
  });

  it("ignores absent or malformed streamed observations", () => {
    expect(toHistoryObservation(undefined)).toBeNull();
    expect(toHistoryObservation({ id: 1, response_time_ms: 0 })).toBeNull();
  });

  it("normalizes canonical and uppercase HTTP/TCP check types", () => {
    const base = historyEntry(observation(1));
    for (const [input, expected] of [
      ["HTTP", "http"],
      ["http", "http"],
      ["TCP", "tcp"],
      ["tcp", "tcp"],
    ] as const) {
      expect(toHistoryObservation({ ...base, check_type: input })?.checkType).toBe(expected);
    }
    expect(toHistoryObservation({ ...base, check_type: "SMTP" })).toBeNull();
  });
});

describe("service history metrics", () => {
  it("renders authoritative KPI values independently of a bounded observation count", () => {
    const window = historyWindow([observation(1), observation(2)], {
      isTruncated: true,
    });
    window.summary = {
      ...window.summary,
      totalChecks: 2_505,
      successfulChecks: 2_004,
      uptimePercent: 80,
      averageLatencyMs: 15,
    };

    const markup = renderToStaticMarkup(<ServiceHistorySummaryMetrics summary={window.summary} />);
    const countMarkup = renderToStaticMarkup(
      <ServiceHistorySuccessCount summary={window.summary} />,
    );
    expect(markup).toContain("80.00%");
    expect(markup).toContain("15 ms");
    expect(countMarkup).toContain("2004 of 2505 checks successful");
    expect(window.observations).toHaveLength(2);
    expect(window.summary.totalChecks).toBe(2_505);
  });

  it("keeps missing failed latency out of chart values while retaining exact zero", () => {
    const points = buildHistoryChartData([
      observation(1, { isSuccess: false, latencyMs: null }),
      observation(2, { isSuccess: false, latencyMs: 0 }),
    ]);
    expect(points.map((point) => point.failureLatency)).toEqual([null, 0]);
  });

  it("formats zero, sub-millisecond, and normal latency values without losing their meaning", () => {
    expect(formatLatency(0)).toBe("0 ms");
    expect(formatLatency(0.5)).toBe("<1 ms");
    expect(formatLatency(12)).toBe("12 ms");
  });

  it("calculates uptime from all observations and average latency from measured successes", () => {
    const metrics = calculateHistoryMetrics([
      observation(1, { latencyMs: 10 }),
      observation(2, { latencyMs: 20 }),
      observation(3, { isSuccess: false, statusCode: 503, latencyMs: 10_001 }),
    ]);

    expect(metrics.uptimePercent).toBeCloseTo(66.67, 2);
    expect(metrics.averageLatencyMs).toBe(15);
    expect(metrics.lastCheckedAt).toBe("2026-08-23T12:00:03Z");
  });

  it("handles empty history and missing latency", () => {
    expect(calculateHistoryMetrics([])).toEqual({
      uptimePercent: null,
      averageLatencyMs: null,
      lastCheckedAt: null,
    });
    expect(
      calculateHistoryMetrics([observation(1, { latencyMs: null })]).averageLatencyMs,
    ).toBeNull();
  });
});

describe("service history rows", () => {
  it("renders HTTP status, latency, and a failed state", () => {
    const markup = renderToStaticMarkup(
      <ServiceHistoryRow
        observation={observation(1, {
          isSuccess: false,
          statusCode: 503,
          latencyMs: 112,
          message: "503 Service Unavailable",
        })}
      />,
    );

    expect(markup).toContain("503 Service Unavailable");
    expect(markup).toContain("112 ms");
    expect(markup).toContain("text-destructive");
  });

  it("renders TCP success without HTTP wording", () => {
    const markup = renderToStaticMarkup(
      <ServiceHistoryRow
        observation={observation(1, {
          checkType: "tcp",
          statusCode: null,
          latencyMs: 11,
          message: "connected",
        })}
      />,
    );

    expect(markup).toContain("Connected");
    expect(markup).toContain("11 ms");
    expect(markup).not.toContain("HTTP");
    expect(markup).not.toContain("No response");
  });

  it("normalizes TCP and HTTP timeout failures differently", () => {
    const tcp = getHistoryRowPresentation(
      observation(1, {
        checkType: "tcp",
        isSuccess: false,
        statusCode: null,
        message: "dial tcp 10.0.0.2:443: i/o timeout",
      }),
    );
    const http = getHistoryRowPresentation(
      observation(2, {
        isSuccess: false,
        statusCode: null,
        message: "context deadline exceeded",
      }),
    );

    expect(tcp.result).toBe("Connection timed out");
    expect(http.result).toBe("Request timed out");
  });

  it("renders a TCP refusal as a prominent normalized failure", () => {
    const markup = renderToStaticMarkup(
      <ServiceHistoryRow
        observation={observation(1, {
          checkType: "tcp",
          isSuccess: false,
          statusCode: null,
          latencyMs: 4,
          message: "dial tcp 10.0.0.2:443: connect: connection refused",
        })}
      />,
    );

    expect(markup).toContain("Connection refused");
    expect(markup).toContain("4 ms");
    expect(markup).toContain("text-destructive");
  });

  it("uses the same concise failure presentation in tooltips", () => {
    const markup = renderToStaticMarkup(
      <ObservationTooltip
        observation={observation(1, {
          isSuccess: false,
          statusCode: 503,
          latencyMs: 84,
          message: "upstream returned internal diagnostic details",
        })}
      />,
    );

    expect(markup).toContain("503 Service Unavailable");
    expect(markup).toContain("84 ms");
    expect(markup).toContain("text-destructive");
    expect(markup).not.toContain("internal diagnostic details");
  });

  it("formats multi-day chart tooltips deterministically without cluttering short ranges", () => {
    expect(formatObservationTooltipTime(observation(1).observedAt, 168, "UTC")).toBe(
      "23 Aug, 12:00:01",
    );
    expect(formatObservationTooltipTime(observation(1).observedAt, 6, "UTC")).toBe("12:00:01");
  });
});

describe("history pagination", () => {
  it("removes duplicate observations while appending cursor pages", () => {
    const firstPage = [observation(3), observation(2)];
    const secondPage = [observation(2), observation(1)];

    const combined = deduplicateHistory([...firstPage, ...secondPage]);

    expect(combined.map((item) => item.id)).toEqual([3, 2, 1]);
  });

  it("renders deduplicated appended checks inside the bounded history container", () => {
    const combined = deduplicateHistory([
      observation(3),
      observation(2),
      observation(2),
      observation(1),
    ]);
    const markup = renderToStaticMarkup(<RecentChecksList observations={combined} />);

    expect(markup).toContain('aria-label="Recent checks history"');
    expect(markup).toContain("max-h-[480px]");
    expect(markup).toContain("overflow-y-auto");
    expect(markup.match(/data-history-observation-id=/g)).toHaveLength(3);
  });

  it("does not render the loaded item count as a recent-check total", () => {
    const markup = renderToStaticMarkup(<RecentChecksHeading />);

    expect(markup).toContain("Recent Checks");
    expect(markup).toMatch(/Recent Checks<\/h2><\/div><\/div>$/);
  });

  it("keeps loaded pages after load-more failure and succeeds when retried", async () => {
    let nextPageAttempts = 0;
    const firstPage = historyPage([3, 2], "cursor-1");
    const secondPage = historyPage([1]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const observer = new InfiniteQueryObserver(client, {
      queryKey: serviceHistoryKeys.list(7, 20),
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

    const failedLoadMore = observer.getCurrentResult();
    expect(failedLoadMore.isFetchNextPageError).toBe(true);
    expect(failedLoadMore.data?.pages).toEqual([firstPage]);
    const errorMarkup = renderToStaticMarkup(
      <RecentChecksPagination
        hasNextPage={true}
        isFetchingNextPage={false}
        isFetchNextPageError={true}
        onLoadMore={() => {}}
      />,
    );
    expect(errorMarkup).toContain("Couldn’t load older checks.");
    expect(errorMarkup).toContain("Try again");

    await observer.fetchNextPage();

    expect(nextPageAttempts).toBe(2);
    expect(observer.getCurrentResult().isFetchNextPageError).toBe(false);
    expect(observer.getCurrentResult().data?.pages).toEqual([firstPage, secondPage]);

    unsubscribe();
    client.clear();
  });
});
