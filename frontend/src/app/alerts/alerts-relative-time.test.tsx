// @vitest-environment happy-dom

import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ActiveAlertRow, AlertHistoryTable } from "@/app/alerts/alerts-page.tsx";
import { alertsApi } from "@/data/alerts.ts";
import type { Alert } from "@/types/alert.ts";

const roots: Root[] = [];

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
    title: "Node is offline",
    message: "Connection refused",
    status: "active",
    triggered_at: "2026-08-23T10:00:00Z",
    resolved_at: null,
    closed_at: null,
    closure_reason: null,
    ...overrides,
  };
}

function renderLive(node: ReactNode) {
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  roots.push(root);
  act(() => root.render(node));
  return container;
}

async function advanceOneSecond() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1_000);
  });
}

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
  document.body.replaceChildren();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe("alert relative times", () => {
  it("advances an active alert's triggered time without a query refetch", async () => {
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T10:00:05Z");
    const activeRequest = vi.spyOn(alertsApi, "listActive");
    const historyRequest = vi.spyOn(alertsApi, "listHistory");
    const container = renderLive(<ActiveAlertRow alert={alert()} />);

    expect(container.textContent).toContain("Triggered 5 seconds ago");

    await advanceOneSecond();

    expect(container.textContent).toContain("Triggered 6 seconds ago");
    expect(activeRequest).not.toHaveBeenCalled();
    expect(historyRequest).not.toHaveBeenCalled();
  });

  it("advances recovered relative time", async () => {
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T10:01:05Z");
    const container = renderLive(
      <AlertHistoryTable
        alerts={[
          alert({
            status: "resolved",
            resolved_at: "2026-08-23T10:01:00Z",
          }),
        ]}
      />,
    );

    expect(container.textContent).toContain("Recovered 5 seconds ago");

    await advanceOneSecond();

    expect(container.textContent).toContain("Recovered 6 seconds ago");
  });

  it("advances administrative closure relative time while preserving its reason", async () => {
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T10:01:05Z");
    const container = renderLive(
      <AlertHistoryTable
        alerts={[
          alert({
            status: "closed",
            closed_at: "2026-08-23T10:01:00Z",
            closure_reason: "rule_disabled",
          }),
        ]}
      />,
    );

    expect(container.textContent).toContain("Rule disabled 5 seconds ago");

    await advanceOneSecond();

    expect(container.textContent).toContain("Rule disabled 6 seconds ago");
  });

  it("keeps resolved history duration fixed while relative times advance", async () => {
    vi.useFakeTimers();
    vi.setSystemTime("2026-08-23T10:01:05Z");
    const container = renderLive(
      <AlertHistoryTable
        alerts={[
          alert({
            status: "resolved",
            triggered_at: "2026-08-23T10:00:18Z",
            resolved_at: "2026-08-23T10:01:00Z",
          }),
        ]}
      />,
    );
    const durationCount = container.textContent?.match(/42s/g)?.length;

    await advanceOneSecond();

    expect(container.textContent?.match(/42s/g)?.length).toBe(durationCount);
    expect(container.textContent).toContain("Recovered 6 seconds ago");
  });
});

describe("alert history status alignment", () => {
  it("centers only terminal status badges at a consistent minimum width", () => {
    const container = renderLive(
      <AlertHistoryTable
        alerts={[
          alert({ id: 1, status: "resolved", resolved_at: "2026-08-23T10:01:00Z" }),
          alert({
            id: 2,
            status: "closed",
            closed_at: "2026-08-23T10:01:00Z",
            closure_reason: "rule_disabled",
          }),
          alert({ id: 3 }),
        ]}
      />,
    );
    const badge = (text: string) =>
      [...container.querySelectorAll("div")].find((element) => element.textContent === text);

    expect(badge("resolved")?.className).toContain("min-w-18");
    expect(badge("resolved")?.className).toContain("justify-center");
    expect(badge("closed")?.className).toContain("min-w-18");
    expect(badge("closed")?.className).toContain("justify-center");
    expect(badge("active")?.className).not.toContain("min-w-18");
    expect(badge("resolved")?.parentElement?.className).toContain("justify-end");
    expect(badge("closed")?.parentElement?.className).toContain("justify-end");
  });
});
