import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import {
  alertRuleTargetLabels,
  buildAlertRuleCreatePayload,
  buildAlertRuleUpdatePayload,
  showsSpecificResourceSelector,
  type AlertRuleFormValues,
  alertRuleTargetLabel,
} from "@/app/alerts/alert-rule-targeting.ts";
import { AlertsPageContent } from "@/app/alerts/alerts-page.tsx";
import { settingsTabFromSearchParams } from "@/app/settings/settings-tabs.ts";
import { alertHistoryKeys } from "@/hooks/use-alerts.ts";
import type { AlertPage, AlertRule, RuleType } from "@/types/alert.ts";

function formValues(overrides: Partial<AlertRuleFormValues> = {}): AlertRuleFormValues {
  return {
    ruleType: "service_unhealthy",
    targetScope: "all",
    nodeId: "",
    serviceId: "",
    thresholdValue: "",
    consecutiveFailures: "3",
    severity: "warning",
    isEnabled: true,
    ...overrides,
  };
}

function rule(overrides: Partial<AlertRule> = {}): AlertRule {
  return {
    id: 1,
    project_id: 7,
    node_id: null,
    service_id: null,
    rule_type: "service_unhealthy",
    threshold_value: null,
    severity: "warning",
    is_enabled: true,
    target_scope: "all",
    created_at: "2026-08-27T10:00:00Z",
    updated_at: "2026-08-27T10:00:00Z",
    ...overrides,
  };
}

function emptyPage(): AlertPage {
  return {
    alerts: [],
    pagination: { limit: 25, hasMore: false, nextCursor: null },
  };
}

function renderLoadedEmptyAlerts(rules: AlertRule[], role: "owner" | "admin" | "viewer") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(alertHistoryKeys.active(7), emptyPage());
  client.setQueryData(alertHistoryKeys.history(7, {}, 25), {
    pages: [emptyPage()],
    pageParams: [null],
  });
  client.setQueryData(["alert-rules", 7], { alert_rules: rules });
  client.setQueryData(["projects", 7], { project: { current_user_role: role } });

  const markup = renderToStaticMarkup(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/alerts"]}>
        <AlertsPageContent activeProjectId={7} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  client.clear();
  return markup;
}

describe("alert rule targeting form", () => {
  it("uses clear service and node scope labels and a conditional resource selector", () => {
    expect(alertRuleTargetLabels("service_unhealthy")).toEqual({
      all: "All services",
      specific: "Specific service",
      resource: "Service",
    });
    expect(alertRuleTargetLabels("cpu_above_threshold")).toEqual({
      all: "All nodes",
      specific: "Specific node",
      resource: "Node",
    });
    expect(showsSpecificResourceSelector({ targetScope: "all" })).toBe(false);
    expect(showsSpecificResourceSelector({ targetScope: "specific" })).toBe(true);
  });

  it.each([
    ["service_unhealthy", "all", "", null, null],
    ["service_unhealthy", "specific", "13", null, 13],
    ["node_offline", "all", "", null, null],
    ["node_offline", "specific", "4", 4, null],
  ] as const)(
    "builds %s %s create and update payloads with nullable target IDs",
    (ruleType, targetScope, resourceId, expectedNodeId, expectedServiceId) => {
      const values = formValues({
        ruleType: ruleType as RuleType,
        targetScope,
        nodeId: ruleType === "service_unhealthy" ? "" : resourceId,
        serviceId: ruleType === "service_unhealthy" ? resourceId : "",
      });

      expect(buildAlertRuleCreatePayload(values, 7)).toMatchObject({
        project_id: 7,
        target_scope: targetScope,
        node_id: expectedNodeId,
        service_id: expectedServiceId,
      });
      expect(buildAlertRuleUpdatePayload(values)).toMatchObject({
        target_scope: targetScope,
        node_id: expectedNodeId,
        service_id: expectedServiceId,
      });
    },
  );
});

describe("alert rule list targeting", () => {
  it("shows All targets or the selected resource name without opening the editor", () => {
    expect(alertRuleTargetLabel(rule(), [], [])).toBe("All services");
    expect(
      alertRuleTargetLabel(
        rule({ target_scope: "specific", service_id: 13 }),
        [],
        [{ id: 13, name: "Landing" }],
      ),
    ).toBe("Landing");
    expect(
      alertRuleTargetLabel(
        rule({
          rule_type: "cpu_above_threshold",
          target_scope: "all",
          threshold_value: 80,
        }),
        [],
        [],
      ),
    ).toBe("All nodes");
  });
});

describe("alerts rule discoverability", () => {
  it("shows setup CTAs when no enabled rules exist", () => {
    const markup = renderLoadedEmptyAlerts([], "admin");

    expect(markup).toContain("Manage rules");
    expect(markup).toContain("No alert rules configured");
    expect(markup).toContain("Create alert rule");
    expect(markup).toContain("/settings?tab=alert-rules");
  });

  it("uses the normal empty state when enabled rules exist but none have triggered", () => {
    const markup = renderLoadedEmptyAlerts([rule()], "admin");

    expect(markup).toContain("No alerts");
    expect(markup).toContain("Nothing has triggered your configured alert rules.");
    expect(markup).not.toContain("No alert rules configured");
    expect(markup).not.toContain("Create alert rule");
  });

  it("does not offer a viewer a rule-creation action", () => {
    const markup = renderLoadedEmptyAlerts([], "viewer");

    expect(markup).toContain("View rules");
    expect(markup).not.toContain("Manage rules");
    expect(markup).not.toContain("Create alert rule");
  });

  it("selects the Alert Rules settings tab from a stable deep link", () => {
    expect(settingsTabFromSearchParams(new URLSearchParams("tab=alert-rules"), false)).toBe(
      "alert-rules",
    );
    expect(settingsTabFromSearchParams(new URLSearchParams("tab=members"), false)).toBe("members");
    expect(settingsTabFromSearchParams(new URLSearchParams("tab=unknown"), false)).toBe(
      "notifications",
    );
  });
});
