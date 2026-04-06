import { useState } from "react";
import { Button } from "@/components/ui/button.tsx";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs.tsx";
import { useAlertRules, useAlerts, useUpdateAlertRule } from "@/hooks/use-alerts.ts";
import { formatDate } from "@/lib/utils.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { CreateAlertRuleDialog } from "./components/create-alert-rule-dialog.tsx";
import { AnimateIcon, BellIcon, PlusIcon } from "@/components/animate-ui/icons";

export function AlertsPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const { data: alertsData, isLoading: alertsLoading } = useAlerts(activeProjectId ?? 0);
  const { data: rulesData, isLoading: rulesLoading } = useAlertRules(activeProjectId ?? 0);
  const updateRule = useUpdateAlertRule(activeProjectId ?? 0);
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Alerts"
        description="Active alerts and alert rules"
        actions={
          <AnimateIcon asChild animateOnHover>
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <PlusIcon size={14} className="mr-1.5" />
              Add rule
            </Button>
          </AnimateIcon>
        }
      />

      <Tabs defaultValue="active">
        <TabsList>
          <TabsTrigger value="active">Active alerts</TabsTrigger>
          <TabsTrigger value="rules">Alert rules</TabsTrigger>
        </TabsList>

        <TabsContent value="active" className="mt-4">
          {alertsLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}
            </div>
          ) : !alertsData?.alerts?.length ? (
            <EmptyState icon={BellIcon} title="No active alerts" description="All clear." />
          ) : (
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/40">
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Title</th>
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Message</th>
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Status</th>
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Triggered</th>
                  </tr>
                </thead>
                <tbody>
                  {alertsData.alerts.map((alert) => (
                    <tr key={alert.id} className="border-b border-border last:border-0">
                      <td className="px-4 py-2.5 font-medium">{alert.title}</td>
                      <td className="px-4 py-2.5 text-muted-foreground">{alert.message}</td>
                      <td className="px-4 py-2.5">
                        <StatusBadge status={alert.status} />
                      </td>
                      <td className="px-4 py-2.5 text-muted-foreground">{formatDate(alert.triggered_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </TabsContent>

        <TabsContent value="rules" className="mt-4">
          {rulesLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}
            </div>
          ) : !rulesData?.alertRules?.length ? (
            <EmptyState
              icon={BellIcon}
              title="No alert rules"
              description="Create a rule to get notified when something goes wrong."
              action={{ label: "Add rule", onClick: () => setCreateOpen(true) }}
            />
          ) : (
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/40">
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Rule type</th>
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Threshold</th>
                    <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Enabled</th>
                  </tr>
                </thead>
                <tbody>
                  {rulesData.alertRules.map((rule) => (
                    <tr key={rule.id} className="border-b border-border last:border-0">
                      <td className="px-4 py-2.5 font-medium capitalize">{rule.rule_type.replaceAll("_", " ")}</td>
                      <td className="px-4 py-2.5 text-muted-foreground">
                        {rule.threshold_value != null ? `${rule.threshold_value}%` : "—"}
                      </td>
                      <td className="px-4 py-2.5">
                        <Switch
                          checked={rule.is_enabled}
                          onCheckedChange={(checked) =>
                            updateRule.mutate({ id: rule.id, payload: { is_enabled: checked } })
                          }
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </TabsContent>
      </Tabs>

      {activeProjectId && (
        <CreateAlertRuleDialog
          projectId={activeProjectId}
          open={createOpen}
          onOpenChange={setCreateOpen}
        />
      )}
    </div>
  );
}
