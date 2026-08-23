import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { servicesApi } from "@/data/services.ts";
import type { CheckType, ServiceHistoryObservation } from "@/types/service.ts";
import type { ServiceHistoryWindow } from "@/types/service.ts";
import { serviceHistoryWindowQueryOptions } from "@/hooks/use-services.ts";
import {
  ObservationTooltip,
  RecentChecksHeading,
  RecentChecksList,
  ServiceHealthLoadingState,
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
});
