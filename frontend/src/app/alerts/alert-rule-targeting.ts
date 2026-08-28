import type {
  AlertRule,
  AlertRuleCreateInput,
  AlertRuleTargetScope,
  AlertRuleUpdateInput,
  RuleType,
  Severity,
} from "@/types/alert.ts";

export type AlertRuleFormValues = {
  ruleType: string;
  targetScope: AlertRuleTargetScope;
  nodeId?: string;
  serviceId?: string;
  thresholdValue?: string;
  consecutiveFailures?: string;
  severity: Severity;
  isEnabled: boolean;
};

export function ruleToFormValues(rule: AlertRule): AlertRuleFormValues {
  return {
    ruleType: rule.rule_type,
    targetScope: rule.target_scope,
    nodeId: rule.node_id != null ? rule.node_id.toString() : "",
    serviceId: rule.service_id != null ? rule.service_id.toString() : "",
    thresholdValue: rule.threshold_value != null ? rule.threshold_value.toString() : "",
    consecutiveFailures: "3",
    severity: rule.severity,
    isEnabled: rule.is_enabled,
  };
}

export function buildAlertRuleTargetPayload(values: AlertRuleFormValues): {
  target_scope: AlertRuleTargetScope;
  node_id: number | null;
  service_id: number | null;
} {
  const targetsService = values.ruleType === "service_unhealthy";
  const isSpecific = values.targetScope === "specific";
  return {
    target_scope: values.targetScope,
    node_id: isSpecific && !targetsService && values.nodeId ? Number(values.nodeId) : null,
    service_id: isSpecific && targetsService && values.serviceId ? Number(values.serviceId) : null,
  };
}

export function buildAlertRuleCreatePayload(
  values: AlertRuleFormValues,
  projectId: number,
): AlertRuleCreateInput {
  return {
    project_id: projectId,
    rule_type: values.ruleType as RuleType,
    ...buildAlertRuleTargetPayload(values),
    threshold_value: values.thresholdValue ? Number(values.thresholdValue) : null,
    severity: values.severity,
  };
}

export function buildAlertRuleUpdatePayload(values: AlertRuleFormValues): AlertRuleUpdateInput {
  return {
    ...buildAlertRuleTargetPayload(values),
    severity: values.severity,
    is_enabled: values.isEnabled,
    threshold_value: values.thresholdValue ? Number(values.thresholdValue) : null,
  };
}

export function alertRuleTargetLabels(ruleType: RuleType): {
  all: string;
  specific: string;
  resource: string;
} {
  if (ruleType === "service_unhealthy") {
    return { all: "All services", specific: "Specific service", resource: "Service" };
  }
  return { all: "All nodes", specific: "Specific node", resource: "Node" };
}

export function showsSpecificResourceSelector(values: Pick<AlertRuleFormValues, "targetScope">) {
  return values.targetScope === "specific";
}

export function alertRuleTargetLabel(
  rule: AlertRule,
  nodes: Array<{ id: number; name: string }>,
  services: Array<{ id: number; name: string }>,
): string {
  if (rule.rule_type === "service_unhealthy") {
    if (rule.target_scope === "all" || rule.service_id == null) return "All services";
    return (
      services.find((service) => service.id === rule.service_id)?.name ??
      `Service ${rule.service_id}`
    );
  }

  if (rule.target_scope === "all" || rule.node_id == null) return "All nodes";
  return nodes.find((node) => node.id === rule.node_id)?.name ?? `Node ${rule.node_id}`;
}
