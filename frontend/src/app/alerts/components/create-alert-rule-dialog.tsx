import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
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
import { useCreateAlertRule } from "@/hooks/use-alerts.ts";
import type { RuleType } from "@/types/alert.ts";

const RULE_TYPES: { value: RuleType; label: string; hasThreshold: boolean }[] = [
  { value: "node_offline", label: "Node offline", hasThreshold: false },
  { value: "service_unhealthy", label: "Service unhealthy", hasThreshold: false },
  { value: "cpu_above_threshold", label: "CPU above threshold", hasThreshold: true },
  { value: "memory_above_threshold", label: "Memory above threshold", hasThreshold: true },
  { value: "disk_above_threshold", label: "Disk above threshold", hasThreshold: true },
];

const schema = z.object({
  ruleType: z.string().min(1, "Rule type is required"),
  thresholdValue: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

type Props = {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CreateAlertRuleDialog({ projectId, open, onOpenChange }: Props) {
  const createRule = useCreateAlertRule(projectId);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { ruleType: "", thresholdValue: "" },
  });

  const selectedRuleType = form.watch("ruleType");
  const ruleConfig = RULE_TYPES.find((r) => r.value === selectedRuleType);

  async function onSubmit(values: FormValues) {
    await createRule.mutateAsync({
      project_id: projectId,
      rule_type: values.ruleType as RuleType,
      threshold_value: values.thresholdValue ? Number(values.thresholdValue) : null,
    });
    toast.success("Alert rule created");
    onOpenChange(false);
    form.reset();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create alert rule</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="ruleType"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Rule type</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select rule type" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {RULE_TYPES.map((r) => (
                        <SelectItem key={r.value} value={r.value}>
                          {r.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            {ruleConfig?.hasThreshold && (
              <FormField
                control={form.control}
                name="thresholdValue"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Threshold (%)</FormLabel>
                    <FormControl>
                      <Input type="number" min="0" max="100" placeholder="e.g. 90" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <div className="flex justify-end gap-2">
              <Button variant="outline" type="button" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createRule.isPending}>
                {createRule.isPending ? "Creating..." : "Create"}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
