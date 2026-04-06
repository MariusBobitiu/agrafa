import { zodResolver } from "@hookform/resolvers/zod";
import { PlusIcon, TrashIcon } from "lucide-react";
import { useForm } from "react-hook-form";
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
import { Switch } from "@/components/ui/switch.tsx";
import { notificationsApi } from "@/data/notifications.ts";

const schema = z.object({ email: z.string().email("Enter a valid email") });
type FormValues = z.infer<typeof schema>;

export function NotificationRecipientsSection({ projectId }: { projectId: number }) {
  const qc = useQueryClient();

  const { data } = useQuery({
    queryKey: ["notifications", projectId],
    queryFn: () => notificationsApi.listRecipients(projectId),
  });

  const create = useMutation({
    mutationFn: (email: string) =>
      notificationsApi.createRecipient({ project_id: projectId, channel: "email", target: email }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications", projectId] }),
  });

  const toggle = useMutation({
    mutationFn: ({ id, is_enabled }: { id: number; is_enabled: boolean }) =>
      notificationsApi.updateRecipient(id, { is_enabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications", projectId] }),
  });

  const remove = useMutation({
    mutationFn: (id: number) => notificationsApi.deleteRecipient(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notifications", projectId] }),
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "" },
  });

  async function onSubmit(values: FormValues) {
    await create.mutateAsync(values.email);
    toast.success("Recipient added");
    form.reset();
  }

  return (
    <div className="space-y-4 max-w-lg">
      <div>
        <h2 className="text-sm font-semibold">Email notifications</h2>
        <p className="text-sm text-muted-foreground">
          Add email addresses to receive alert notifications.
        </p>
      </div>

      {data?.recipients?.map((r) => (
        <div key={r.id} className="flex items-center justify-between rounded-md border border-border px-3 py-2">
          <span className="text-sm">{r.target}</span>
          <div className="flex items-center gap-2">
            <Switch
              checked={r.is_enabled}
              onCheckedChange={(checked) => toggle.mutate({ id: r.id, is_enabled: checked })}
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-muted-foreground hover:text-destructive"
              onClick={() => remove.mutate(r.id)}
            >
              <TrashIcon size={14} />
            </Button>
          </div>
        </div>
      ))}

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-2">
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input type="email" placeholder="new@example.com" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button type="submit" size="sm" disabled={create.isPending}>
            <PlusIcon size={14} className="mr-1" />
            Add
          </Button>
        </form>
      </Form>
    </div>
  );
}
