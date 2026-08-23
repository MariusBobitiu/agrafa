import {
  InfiniteQueryObserver,
  QueryClient,
  QueryObserver,
  type InfiniteData,
} from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { servicesApi } from "@/data/services.ts";
import type { CheckType, ServiceHistoryObservation, ServiceHistoryPage } from "@/types/service.ts";
import type { ServiceHistoryWindow } from "@/types/service.ts";
import {
  refreshServiceHistoryLatestPage,
  serviceHistoryKeys,
  serviceHistoryWindowQueryOptions,
} from "@/hooks/use-services.ts";
import {
  ObservationTooltip,
  RecentChecksHeading,
  RecentChecksList,
  RecentChecksPagination,
  ServiceHealthLoadingState,
  ServiceHistoryRefreshError,
  ServiceHistoryRow,
} from "./components/service-history.tsx";
import {
  calculateHistoryMetrics,
  deduplicateHistory,
  formatLatency,
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
    observedAt: `2026-08-23T12:00:${String(id).padStart(2, "0")}Z`,
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

afterEach(() => {
  vi.restoreAllMocks();
});

describe("service history loading states", () => {
  it("is pending without data on first load and retains the previous range while refetching", async () => {
    const firstWindow: ServiceHistoryWindow = {
      observations: [observation(1)],
      isTruncated: false,
    };
    const secondWindow: ServiceHistoryWindow = {
      observations: [observation(2)],
      isTruncated: false,
    };
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
    const staleWindow: ServiceHistoryWindow = {
      observations: [observation(1)],
      isTruncated: false,
    };
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
});

describe("service history SSE refresh", () => {
  it("replaces only the newest active list page and preserves every loaded older page", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const key = serviceHistoryKeys.list(7, 20);
    const olderPage = historyPage([40, 39], "cursor-2");
    const oldestPage = historyPage([20, 19]);
    const cached: InfiniteData<ServiceHistoryPage, string | null> = {
      pages: [historyPage([60, 59], "cursor-1"), olderPage, oldestPage],
      pageParams: [null, "cursor-1", "cursor-2"],
    };
    client.setQueryData(key, cached);
    const observer = new QueryObserver(client, {
      queryKey: key,
      queryFn: () => Promise.resolve(cached),
      staleTime: Infinity,
    });
    const unsubscribe = observer.subscribe(() => {});
    const latestPage = historyPage([61, 60], "cursor-1");
    const historyRequest = vi.spyOn(servicesApi, "history").mockResolvedValue(latestPage);
    const invalidate = vi.spyOn(client, "invalidateQueries");

    await refreshServiceHistoryLatestPage(client, 7);

    const refreshed = client.getQueryData<InfiniteData<ServiceHistoryPage, string | null>>(key);
    expect(historyRequest).toHaveBeenCalledTimes(1);
    expect(historyRequest).toHaveBeenCalledWith(7, { limit: 20 });
    expect(invalidate).not.toHaveBeenCalled();
    expect(refreshed?.pages[0]).toEqual(latestPage);
    expect(refreshed?.pages[1]).toBe(olderPage);
    expect(refreshed?.pages[2]).toBe(oldestPage);
    expect(refreshed?.pageParams).toBe(cached.pageParams);

    unsubscribe();
    client.clear();
  });

  it("does not refetch or modify any range query", async () => {
    const client = new QueryClient();
    const rangeKey = serviceHistoryKeys.window(7, 168);
    const rangeData: ServiceHistoryWindow = {
      observations: [observation(1)],
      isTruncated: false,
    };
    client.setQueryData(rangeKey, rangeData);
    const rangeRequest = vi.spyOn(servicesApi, "historyWindow");

    await refreshServiceHistoryLatestPage(client, 7);

    expect(rangeRequest).not.toHaveBeenCalled();
    expect(client.getQueryData(rangeKey)).toBe(rangeData);
    client.clear();
  });
});

describe("service history metrics", () => {
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

  it("includes the date in multi-day chart tooltips without cluttering short ranges", () => {
    const multiDay = renderToStaticMarkup(
      <ObservationTooltip observation={observation(1)} rangeHours={168} />,
    );
    const shortRange = renderToStaticMarkup(
      <ObservationTooltip observation={observation(1)} rangeHours={6} />,
    );

    expect(multiDay).toMatch(/23 Aug, \d{2}:00:01/);
    expect(shortRange).toMatch(/\d{2}:00:01/);
    expect(shortRange).not.toContain("23 Aug");
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
