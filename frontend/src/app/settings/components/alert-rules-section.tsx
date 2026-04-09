import { useState } from "react";
import { PlusIcon, TrashIcon } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import { ConfirmDialog } from "@/components/ui/confirm-dialog.tsx";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { useAlertRules, useUpdateAlertRule, useDeleteAlertRule } from "@/hooks/use-alerts.ts";
import { useCanWrite } from "@/hooks/use-project-role.ts";
import { CreateAlertRuleDialog } from "../../alerts/components/create-alert-rule-dialog.tsx";
import { BellIcon } from "@/components/animate-ui/icons";
import type { AlertRule, RuleType, Severity } from "@/types/alert.ts";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatRuleType(ruleType: RuleType): string {
  return ruleType
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

function ruleCategory(ruleType: RuleType): string {
  switch (ruleType) {
    case "node_offline":
      return "Node";
    case "service_unhealthy":
      return "Service";
    case "cpu_above_threshold":
    case "memory_above_threshold":
    case "disk_above_threshold":
      return "Metric";
    default:
      return "Rule";
  }
}

const severityClasses: Record<Severity, string> = {
  critical: "bg-red-500/15 text-red-400 border-red-500/20",
  warning: "bg-yellow-500/15 text-yellow-400 border-yellow-500/20",
  info: "bg-blue-500/15 text-blue-400 border-blue-500/20",
};

// ─── Row ──────────────────────────────────────────────────────────────────────

function AlertRuleRow({
  rule,
  onToggle,
  onDelete,
  canWrite,
}: {
  rule: AlertRule;
  onToggle: (id: number, enabled: boolean) => void;
  onDelete: (id: number) => void;
  canWrite: boolean;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <>
      <div className="flex items-center gap-4 rounded-xl border border-border px-5 py-3.5 bg-card hover:bg-muted/30 transition-colors">
        {/* ── Identity ── */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-foreground leading-snug">
            {formatRuleType(rule.rule_type)}
          </p>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-xs text-muted-foreground/70">
              {ruleCategory(rule.rule_type)}
              {rule.threshold_value !== null && ` · threshold > ${rule.threshold_value}%`}
            </span>
            {rule.severity && (
              <span className={`inline-flex items-center rounded border px-1.5 py-0 text-[10px] font-medium leading-4 capitalize ${severityClasses[rule.severity]}`}>
                {rule.severity}
              </span>
            )}
          </div>
        </div>

        {/* ── Actions ── */}
        <div className="flex items-center gap-3 shrink-0">
          <Switch
            checked={rule.is_enabled}
            onCheckedChange={(checked) => canWrite && onToggle(rule.id, checked)}
            disabled={!canWrite}
          />
          {canWrite && (
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-muted-foreground/50 hover:text-destructive"
              onClick={() => setDeleteOpen(true)}
            >
              <TrashIcon size={14} />
            </Button>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete rule"
        description="This will permanently delete the alert rule. Future alerts from this rule will no longer be triggered."
        onConfirm={() => {
          onDelete(rule.id);
          setDeleteOpen(false);
        }}
      />
    </>
  );
}

// ─── Section ──────────────────────────────────────────────────────────────────

export function AlertRulesSection({ projectId }: { projectId: number }) {
  const [createOpen, setCreateOpen] = useState(false);
  const canWrite = useCanWrite(projectId);

  const { data, isLoading } = useAlertRules(projectId);
  const toggle = useUpdateAlertRule(projectId);
  const remove = useDeleteAlertRule(projectId);

  function handleToggle(id: number, enabled: boolean) {
    toggle.mutate({ id, payload: { is_enabled: enabled } });
  }

  function handleDelete(id: number) {
    remove.mutate(id, {
      onSuccess: () => toast.success("Rule deleted"),
    });
  }

  const rules = data?.alert_rules ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Alert rules</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Define when Agrafa should trigger a notification.
          </p>
        </div>
        {canWrite && (
          <Button size="sm" variant={"secondary"} onClick={() => setCreateOpen(true)}>
            <PlusIcon size={14} />
            Add rule
          </Button>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-15 w-full rounded-xl" />
          ))}
        </div>
      ) : rules.length === 0 ? (
        <EmptyState
          icon={BellIcon}
          title="No alert rules"
          description="Add a rule to get notified when something goes wrong."
          action={canWrite ? { label: "Add rule", onClick: () => setCreateOpen(true) } : undefined}
        />
      ) : (
        <div className="space-y-2">
          {rules.map((rule) => (
            <AlertRuleRow
              key={rule.id}
              rule={rule}
              onToggle={handleToggle}
              onDelete={handleDelete}
              canWrite={canWrite}
            />
          ))}
        </div>
      )}

      <CreateAlertRuleDialog
        projectId={projectId}
        open={createOpen}
        onOpenChange={setCreateOpen}
      />
    </div>
  );
}
