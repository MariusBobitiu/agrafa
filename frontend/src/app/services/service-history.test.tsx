import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { CheckType, ServiceHistoryObservation } from "@/types/service.ts";
import { ServiceHistoryRow } from "./components/service-history.tsx";
import {
  calculateHistoryMetrics,
  deduplicateHistory,
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

describe("service history metrics", () => {
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
});

describe("history pagination", () => {
  it("removes duplicate observations while appending cursor pages", () => {
    const firstPage = [observation(3), observation(2)];
    const secondPage = [observation(2), observation(1)];

    const combined = deduplicateHistory([...firstPage, ...secondPage]);

    expect(combined.map((item) => item.id)).toEqual([3, 2, 1]);
  });
});
