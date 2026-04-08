import { useEffect } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { MailIcon, PlusIcon, SaveIcon, TrashIcon } from "lucide-react";
import { useFieldArray, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button.tsx";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
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
import { Separator } from "@/components/ui/separator.tsx";
import { notificationsApi } from "@/data/notifications.ts";
import type { Severity } from "@/types/alert.ts";

// ─── Schema ───────────────────────────────────────────────────────────────────

const rowSchema = z.object({
  target: z.string().email("Enter a valid email"),
  min_severity: z.enum(["info", "warning", "critical"]),
});

const schema = z.object({ recipients: z.array(rowSchema) });
type FormValues = z.infer<typeof schema>;

// ─── Component ────────────────────────────────────────────────────────────────

export function NotificationRecipientsSection({ projectId }: { projectId: number }) {
  const qc = useQueryClient();

  const { data, isSuccess } = useQuery({
    queryKey: ["notifications", projectId],
    queryFn: () => notificationsApi.listRecipients(projectId),
  });

  const save = useMutation({
    mutationFn: (recipients: { target: string; min_severity: Severity }[]) =>
      notificationsApi.setRecipients({ channel_type: "email", project_id: projectId, recipients }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications", projectId] });
      toast.success("Recipients saved");
    },
    onError: () => toast.error("Couldn't save recipients. Try again."),
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { recipients: [] },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "recipients",
  });

  // Populate form once data loads
  useEffect(() => {
    if (!isSuccess) return;
    const loaded = (data?.notification_recipients ?? []).map((r) => ({
      target: r.target,
      min_severity: r.min_severity,
    }));
    form.reset({ recipients: loaded });
  }, [isSuccess, data]);

  async function onSubmit(values: FormValues) {
    await save.mutateAsync(values.recipients);
  }

  const severityLabel: Record<Severity, string> = {
    critical: "Critical",
    warning: "Warning",
    info: "Info",
  };

  return (
    <div className="space-y-6">

      {/* ── Recipients card ── */}
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <div className="rounded-xl border border-border overflow-hidden">

            {/* Header */}
            <div className="flex items-center justify-between px-6 py-5">
              <div>
                <h2 className="text-sm font-semibold">Email recipients</h2>
                <p className="mt-0.5 text-sm text-muted-foreground">
                  Each recipient only receives alerts at or above their minimum severity.
                </p>
              </div>
              {fields.length > 0 && (
                <Button type="submit" size="sm" disabled={save.isPending}>
                  <SaveIcon size={14} />
                  Save
                </Button>
              )}
            </div>

            <Separator />

            {/* List / empty state */}
            {fields.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-2 px-6 py-10 text-center">
                <div className="flex size-10 items-center justify-center rounded-full bg-muted">
                  <MailIcon size={16} className="text-muted-foreground" />
                </div>
                <p className="text-sm font-medium text-foreground">No recipients yet</p>
                <p className="text-xs text-muted-foreground max-w-xs">
                  Add an email address below to start receiving alert notifications.
                </p>
              </div>
            ) : (
              <div className="divide-y divide-border">
                {fields.map((field, index) => (
                  <div key={field.id} className="flex items-start gap-3 px-6 py-3.5">
                    <FormField
                      control={form.control}
                      name={`recipients.${index}.target`}
                      render={({ field: f }) => (
                        <FormItem className="flex-1">
                          <FormControl>
                            <Input
                              type="email"
                              placeholder="name@example.com"
                              className="font-mono text-sm"
                              {...f}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`recipients.${index}.min_severity`}
                      render={({ field: f }) => (
                        <FormItem className="w-32 shrink-0">
                          <Select onValueChange={f.onChange} value={f.value}>
                            <FormControl>
                              <SelectTrigger className="text-sm">
                                <SelectValue />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              {(["critical", "warning", "info"] as Severity[]).map((s) => (
                                <SelectItem key={s} value={s}>
                                  {severityLabel[s]}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      className="mt-1 text-muted-foreground/40 hover:text-destructive shrink-0"
                      onClick={() => remove(index)}
                    >
                      <TrashIcon size={13} />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            <Separator />

            {/* Footer */}
            <div className="px-6 py-4 bg-muted/20">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => append({ target: "", min_severity: "warning" })}
              >
                <PlusIcon size={14} />
                Add recipient
              </Button>
            </div>

          </div>
        </form>
      </Form>

    </div>
  );
}
