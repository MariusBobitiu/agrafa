import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { cn } from "@/lib/utils.ts";
import { useCreateAlertRule } from "@/hooks/use-alerts.ts";
import { useNodes } from "@/hooks/use-nodes.ts";
import { useServices } from "@/hooks/use-services.ts";
import type { RuleType, Severity } from "@/types/alert.ts";

// ─── Rule definitions ─────────────────────────────────────────────────────────

type RuleConfig = {
  value: RuleType;
  label: string;
  hint: string;
  targetNode: boolean;
  targetService: boolean;
  conditionType: "failures" | "threshold";
  thresholdUnit?: string;
  thresholdPlaceholder?: string;
  defaultSeverity: Severity;
};

// function defaultSeverityForRule(ruleType: RuleType): Severity {
// 	switch (ruleType) {
// 		case "node_offline":
// 		case "service_unhealthy":
// 			return "critical";
// 		case "cpu_above_threshold":
// 		case "memory_above_threshold":
// 		case "disk_above_threshold":
// 			return "warning";
// 		default:
// 			return "warning";
// 	}
// }

const RULE_TYPES: RuleConfig[] = [
  {
    value: "node_offline",
    label: "Node offline",
    hint: "Triggers when a node stops sending heartbeats.",
    targetNode: true,
    targetService: false,
    conditionType: "failures",
    defaultSeverity: "critical",
  },
  {
    value: "service_unhealthy",
    label: "Service unhealthy",
    hint: "Triggers when a service health check reports failure.",
    targetNode: false,
    targetService: true,
    conditionType: "failures",
    defaultSeverity: "critical",
  },
  {
    value: "cpu_above_threshold",
    label: "CPU above threshold",
    hint: "Triggers when CPU usage exceeds your defined limit.",
    targetNode: true,
    targetService: false,
    conditionType: "threshold",
    thresholdUnit: "%",
    thresholdPlaceholder: "85",
    defaultSeverity: "warning",
  },
  {
    value: "memory_above_threshold",
    label: "Memory above threshold",
    hint: "Triggers when memory usage exceeds your defined limit.",
    targetNode: true,
    targetService: false,
    conditionType: "threshold",
    thresholdUnit: "%",
    thresholdPlaceholder: "90",
    defaultSeverity: "warning",
  },
  {
    value: "disk_above_threshold",
    label: "Disk above threshold",
    hint: "Triggers when disk usage exceeds your defined limit.",
    targetNode: true,
    targetService: false,
    conditionType: "threshold",
    thresholdUnit: "%",
    thresholdPlaceholder: "80",
    defaultSeverity: "warning",
  },
];

// ─── Schema ───────────────────────────────────────────────────────────────────

const schema = z
  .object({
    ruleType: z.string().min(1, "Select a rule type"),
    nodeId: z.string().optional(),
    serviceId: z.string().optional(),
    thresholdValue: z.string().optional(),
    consecutiveFailures: z.string().optional(),
    severity: z.enum(["info", "warning", "critical"]),
    isEnabled: z.boolean(),
  })
  .superRefine((values, ctx) => {
    const rule = RULE_TYPES.find((r) => r.value === values.ruleType);
    if (!rule) return;
    if (rule.conditionType === "threshold") {
      const v = Number(values.thresholdValue);
      if (!values.thresholdValue || isNaN(v) || v < 1 || v > 100) {
        ctx.addIssue({
          code: "custom",
          path: ["thresholdValue"],
          message: "Enter a value between 1 and 100",
        });
      }
    }
  });

type FormValues = z.infer<typeof schema>;

const DEFAULT_VALUES: FormValues = {
  ruleType: "",
  nodeId: "",
  serviceId: "",
  thresholdValue: "",
  consecutiveFailures: "3",
  severity: "critical",
  isEnabled: true,
};

// ─── Style constants ──────────────────────────────────────────────────────────

// Kills the lime focus ring; keeps border transition only
const quietTrigger = "h-9 text-sm focus:ring-0 focus:ring-offset-0 focus:border-border";

// Muted hover + checked state for dropdown items
const quietItem =
  "focus:bg-muted focus:text-foreground data-[state=checked]:font-medium data-[state=checked]:text-foreground";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function Hint({ children }: { children: React.ReactNode }) {
  return <p className="text-xs text-muted-foreground/70 leading-snug">{children}</p>;
}

function EmptyState({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-md border border-dashed border-border bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
      {children}
    </div>
  );
}

// ─── Dialog ───────────────────────────────────────────────────────────────────

type Props = {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CreateAlertRuleDialog({ projectId, open, onOpenChange }: Props) {
  const createRule = useCreateAlertRule(projectId);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: DEFAULT_VALUES,
  });

  const selectedRuleType = form.watch("ruleType");
  const rule = RULE_TYPES.find((r) => r.value === selectedRuleType);

  const nodesQuery = useNodes(projectId, {
    enabled: open && !!rule?.targetNode,
  });
  const servicesQuery = useServices(projectId);
  const nodes = nodesQuery.data?.nodes ?? [];
  const services = servicesQuery.data?.services ?? [];

  function handleOpenChange(nextOpen: boolean) {
    onOpenChange(nextOpen);
    if (!nextOpen) form.reset(DEFAULT_VALUES);
  }

  async function onSubmit(values: FormValues) {
    try {
      await createRule.mutateAsync({
        project_id: projectId,
        rule_type: values.ruleType as RuleType,
        node_id: values.nodeId && values.nodeId !== "__any__" ? Number(values.nodeId) : null,
        service_id:
          values.serviceId && values.serviceId !== "__any__" ? Number(values.serviceId) : null,
        threshold_value: values.thresholdValue ? Number(values.thresholdValue) : null,
        severity: values.severity,
      });
      toast.success("Alert rule created");
      handleOpenChange(false);
    } catch {
      toast.error("Couldn't create the rule. Try again.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg gap-0 p-0">
        <DialogHeader className="border-b border-border/60 px-5 pb-4 pt-5">
          <DialogTitle className="text-base">Create alert rule</DialogTitle>
          <DialogDescription className="text-sm">
            Define when Agrafa should flag an issue.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="space-y-3 px-5 py-4">
              {/* Rule type */}
              <FormField
                control={form.control}
                name="ruleType"
                render={({ field }) => (
                  <FormItem className="gap-0">
                    <FormLabel className="mb-1.5 text-sm">Rule type</FormLabel>
                    <Select
                      onValueChange={(val) => {
                        field.onChange(val);
                        form.resetField("nodeId");
                        form.resetField("serviceId");
                        form.resetField("thresholdValue");
                      }}
                      value={field.value}
                    >
                      <FormControl>
                        <SelectTrigger className={quietTrigger}>
                          <SelectValue placeholder="Choose what to monitor…" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {RULE_TYPES.map((r) => (
                          <SelectItem key={r.value} value={r.value} className={quietItem}>
                            {r.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {rule && <Hint>{rule.hint}</Hint>}
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Node selector */}
              {rule?.targetNode && (
                <FormField
                  control={form.control}
                  name="nodeId"
                  render={({ field }) => (
                    <FormItem className="gap-0">
                      <FormLabel className="mb-1.5 text-sm">Node</FormLabel>
                      {nodesQuery.isLoading ? (
                        <EmptyState>Loading nodes…</EmptyState>
                      ) : nodes.length === 0 ? (
                        <EmptyState>No nodes in this project.</EmptyState>
                      ) : (
                        <Select onValueChange={field.onChange} value={field.value}>
                          <FormControl>
                            <SelectTrigger className={quietTrigger}>
                              <SelectValue placeholder="Any node" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem
                              value="__any__"
                              className={cn(quietItem, "text-muted-foreground")}
                            >
                              Any node
                            </SelectItem>
                            {nodes.map((node) => (
                              <SelectItem
                                key={node.id}
                                value={node.id.toString()}
                                className={quietItem}
                              >
                                {node.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      )}
                      <Hint>Leave blank to apply to all nodes.</Hint>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Service selector */}
              {rule?.targetService && (
                <FormField
                  control={form.control}
                  name="serviceId"
                  render={({ field }) => (
                    <FormItem className="gap-0">
                      <FormLabel className="mb-1.5 text-sm">Service</FormLabel>
                      {servicesQuery.isLoading ? (
                        <EmptyState>Loading services…</EmptyState>
                      ) : services.length === 0 ? (
                        <EmptyState>No services in this project.</EmptyState>
                      ) : (
                        <Select onValueChange={field.onChange} value={field.value}>
                          <FormControl>
                            <SelectTrigger className={quietTrigger}>
                              <SelectValue placeholder="Any service" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem
                              value="__any__"
                              className={cn(quietItem, "text-muted-foreground")}
                            >
                              Any service
                            </SelectItem>
                            {services.map((service) => (
                              <SelectItem
                                key={service.id}
                                value={service.id.toString()}
                                className={quietItem}
                              >
                                {service.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      )}
                      <Hint>Leave blank to apply to all services.</Hint>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Threshold */}
              {rule?.conditionType === "threshold" && (
                <FormField
                  control={form.control}
                  name="thresholdValue"
                  render={({ field }) => (
                    <FormItem className="gap-0">
                      <FormLabel className="mb-1.5 text-sm">Alert when above</FormLabel>
                      <div className="flex items-center">
                        <FormControl>
                          <Input
                            type="number"
                            min="1"
                            max="100"
                            placeholder={rule.thresholdPlaceholder}
                            className={cn(
                              "h-9 w-24 rounded-r-none text-sm tabular-nums",
                              "[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none",
                            )}
                            {...field}
                          />
                        </FormControl>
                        <span className="flex h-9 select-none items-center rounded-r-md border border-l-0 border-input bg-muted/40 px-3 text-sm text-muted-foreground">
                          {rule.thresholdUnit ?? "%"}
                        </span>
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Consecutive failures */}
              {rule?.conditionType === "failures" && (
                <FormField
                  control={form.control}
                  name="consecutiveFailures"
                  render={({ field }) => (
                    <FormItem className="gap-0">
                      <FormLabel className="mb-1.5 text-sm">Trigger after</FormLabel>
                      <div className="flex items-center gap-2">
                        <FormControl>
                          <Input
                            type="number"
                            min="1"
                            max="20"
                            className={cn(
                              "h-9 w-16 text-center text-sm tabular-nums",
                              "[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none",
                            )}
                            {...field}
                          />
                        </FormControl>
                        <span className="text-sm text-muted-foreground">consecutive checks</span>
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Severity */}
              {rule && (
                <FormField
                  control={form.control}
                  name="severity"
                  render={({ field }) => (
                    <FormItem className="gap-0">
                      <FormLabel className="mb-1.5 text-sm">Severity</FormLabel>
                      <Select onValueChange={field.onChange} value={field.value}>
                        <FormControl>
                          <SelectTrigger className={quietTrigger}>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="critical" className={quietItem}>
                            Critical
                          </SelectItem>
                          <SelectItem value="warning" className={quietItem}>
                            Warning
                          </SelectItem>
                          <SelectItem value="info" className={quietItem}>
                            Info
                          </SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Enable toggle */}
              {rule && (
                <FormField
                  control={form.control}
                  name="isEnabled"
                  render={({ field }) => (
                    <FormItem className="gap-0 pt-0.5">
                      <div className="flex items-center justify-between gap-4">
                        <FormLabel className="mb-0 cursor-pointer text-sm font-medium leading-5">
                          Enable rule
                          <Hint>Start monitoring immediately after creation.</Hint>
                        </FormLabel>

                        <FormControl>
                          <Switch checked={field.value} onCheckedChange={field.onChange} />
                        </FormControl>
                      </div>
                    </FormItem>
                  )}
                />
              )}
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 border-t border-border/60 px-5 py-3">
              <Button
                variant="outline"
                type="button"
                size="sm"
                onClick={() => handleOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" size="sm" disabled={createRule.isPending || !selectedRuleType}>
                {createRule.isPending ? "Creating…" : "Create rule"}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
