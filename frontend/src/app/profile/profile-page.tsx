import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { z } from "zod";
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
import { Separator } from "@/components/ui/separator.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { authApi } from "@/data/auth.ts";
import { useAuth } from "@/hooks/use-auth.ts";
import { SessionsSection } from "@/app/settings/components/sessions-section.tsx";

// ─── Change password schema ───────────────────────────────────────────────────

const passwordSchema = z
  .object({
    current_password: z.string().min(1, "Required"),
    new_password: z.string().min(8, "At least 8 characters"),
    confirm_password: z.string().min(1, "Required"),
  })
  .refine((d) => d.new_password === d.confirm_password, {
    message: "Passwords don't match",
    path: ["confirm_password"],
  });

type PasswordValues = z.infer<typeof passwordSchema>;

// ─── Shared section card shell ────────────────────────────────────────────────

function SectionCard({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-border overflow-hidden">
      {children}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function ProfilePage() {
  const { user } = useAuth();

  const sendVerifyEmail = useMutation({
    mutationFn: () => authApi.sendVerifyEmail(),
    onSuccess: () => toast.success("Verification email sent"),
  });

  const passwordForm = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { current_password: "", new_password: "", confirm_password: "" },
  });

  const changePassword = useMutation({
    mutationFn: (values: PasswordValues) =>
      authApi.changePassword({
        current_password: values.current_password,
        new_password: values.new_password,
      }),
    onSuccess: () => {
      toast.success("Password updated");
      passwordForm.reset();
    },
    onError: () => toast.error("Current password is incorrect"),
  });

  return (
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader title="Profile" />

      <div className="space-y-6">

        {/* ── Profile info section ── */}
        <SectionCard>
          <div className="px-6 py-5">
            <h2 className="text-sm font-semibold">Profile</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              Your account identity and verification status.
            </p>
          </div>
          <Separator />
          <div className="px-6 py-5 space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <p className="text-sm font-medium text-foreground">{user?.name}</p>
                <p className="text-sm text-muted-foreground">{user?.email}</p>
              </div>
              <Badge variant={user?.email_verified ? "default" : "secondary"}>
                {user?.email_verified ? "Verified" : "Unverified"}
              </Badge>
            </div>

            {!user?.email_verified && (
              <div className="flex items-center justify-between rounded-lg border border-border px-4 py-3 bg-muted/20">
                <p className="text-sm text-muted-foreground">Email verification pending</p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => sendVerifyEmail.mutate()}
                  disabled={sendVerifyEmail.isPending}
                >
                  {sendVerifyEmail.isPending ? "Sending…" : "Send link"}
                </Button>
              </div>
            )}
          </div>
        </SectionCard>

        {/* ── Security section ── */}
        <SectionCard>
          <div className="px-6 py-5">
            <h2 className="text-sm font-semibold">Change password</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              Update your account password.
            </p>
          </div>
          <Separator />
          <div className="px-6 py-5">
            <Form {...passwordForm}>
              <form
                onSubmit={passwordForm.handleSubmit((v) => changePassword.mutate(v))}
                className="space-y-4 max-w-md"
              >
                <FormField
                  control={passwordForm.control}
                  name="current_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Current password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="current-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={passwordForm.control}
                  name="new_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>New password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="new-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={passwordForm.control}
                  name="confirm_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Confirm new password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="new-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button type="submit" size="sm" disabled={changePassword.isPending}>
                  {changePassword.isPending ? "Saving…" : "Update password"}
                </Button>
              </form>
            </Form>
          </div>
        </SectionCard>

        {/* ── Sessions section ── */}
        <SectionCard>
          <SessionsSection />
        </SectionCard>

      </div>
    </div>
  );
}
