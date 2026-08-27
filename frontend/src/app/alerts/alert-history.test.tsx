import { InfiniteQueryObserver, QueryClient } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import {
  ActiveAlertRow,
  AlertHistoryPagination,
  AlertHistoryTable,
  EmptyState,
  ErrorState,
  TableSkeleton,
} from "@/app/alerts/alerts-page.tsx";
import { deduplicateAlerts, historyFilterTriggerClass } from "@/app/alerts/alert-presentation.ts";
import { alertHistoryKeys } from "@/hooks/use-alerts.ts";
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
