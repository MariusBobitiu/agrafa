import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { z } from "zod";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import { authApi } from "@/data/auth.ts";
import { useAuth } from "@/hooks/use-auth.ts";
import { useProjectDetail, useUpdateProject } from "@/hooks/use-projects.ts";

const schema = z.object({ name: z.string().min(1, "Name is required") });
type FormValues = z.infer<typeof schema>;

export function GeneralSection({ projectId }: { projectId: number }) {
  const { user } = useAuth();
  const { data } = useProjectDetail(projectId);
  const update = useUpdateProject();
  const sendVerifyEmail = useMutation({
    mutationFn: () => authApi.sendVerifyEmail(),
    onSuccess: () => {
      toast.success("Verification email sent");
    },
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  });

  useEffect(() => {
    if (data?.project.name) form.reset({ name: data.project.name });
  }, [data, form]);

  async function onSubmit(values: FormValues) {
    await update.mutateAsync({ id: projectId, payload: { name: values.name } });
    toast.success("Project updated");
  }

  return (
    <div className="space-y-6">
      <div className="space-y-4 rounded-lg border border-border p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-sm font-semibold">Account</h2>
            <p className="text-sm text-muted-foreground">Manage your email verification status.</p>
          </div>
          <Badge variant={user?.email_verified ? "default" : "secondary"}>
            {user?.email_verified ? "Verified" : "Unverified"}
          </Badge>
        </div>

        <div className="space-y-1">
          <p className="text-sm font-medium">{user?.name}</p>
          <p className="text-sm text-muted-foreground">{user?.email}</p>
        </div>

        {!user?.email_verified && (
          <Alert>
            <AlertTitle>Email verification pending</AlertTitle>
            <AlertDescription>
              Send a fresh verification link if the original email has expired or gone missing.
            </AlertDescription>
          </Alert>
        )}

        {!user?.email_verified && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => sendVerifyEmail.mutate()}
            disabled={sendVerifyEmail.isPending}
          >
            {sendVerifyEmail.isPending ? "Sending..." : "Send verification email"}
          </Button>
        )}
      </div>

      <div>
        <div>
          <h2 className="text-sm font-semibold">Project</h2>
          <p className="text-sm text-muted-foreground">Update your project settings.</p>
        </div>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="mt-4 max-w-sm space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Project name</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" size="sm" disabled={update.isPending}>
              {update.isPending ? "Saving..." : "Save"}
            </Button>
          </form>
        </Form>
      </div>
    </div>
  );
}
