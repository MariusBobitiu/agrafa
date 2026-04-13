import { useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Button } from "@/components/ui/button.tsx";
import { authApi } from "@/data/auth.ts";
import { useAuth } from "@/hooks/use-auth.ts";
import type { User } from "@/types/auth.ts";
import { useMeta } from "@/hooks/use-meta";

export function VerifyEmailPage() {
  useMeta({
    title: "Verify Email",
    description: "Confirm your email address to access all features of Agrafa",
  });
  const { isAuthenticated, refreshUser, user, logout } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const handledSuccessRef = useRef(false);
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token")?.trim() ?? "";
  const continueTo = isAuthenticated
    ? user?.onboarding_completed
      ? "/overview"
      : "/onboarding"
    : "/login";

  const confirm = useQuery({
    queryKey: ["auth", "verify-email", token],
    queryFn: () => authApi.confirmVerifyEmail({ token }),
    enabled: Boolean(token),
    retry: false,
    staleTime: Infinity,
  });

  const resend = useMutation({
    mutationFn: () => authApi.sendVerifyEmail(),
  });

  useEffect(() => {
    if (!confirm.isSuccess || handledSuccessRef.current) return;

    handledSuccessRef.current = true;

    if (isAuthenticated && user) {
      queryClient.setQueryData<{ user: User } | null>(["auth", "me"], (current) =>
        current
          ? {
              user: {
                ...current.user,
                email_verified: true,
              },
            }
          : current,
      );

      void refreshUser().catch(() => undefined);
      navigate(continueTo, { replace: true });
    }
  }, [confirm.isSuccess, continueTo, isAuthenticated, navigate, queryClient, refreshUser, user]);

  if (!token) {
    if (isAuthenticated) {
      return (
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Verify your email</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Confirm {user?.email} before continuing in Agrafa.
            </p>
          </div>

          <Alert>
            <AlertTitle>Check your inbox</AlertTitle>
            <AlertDescription>
              We sent a verification link to your email address. Open that link to unlock the rest
              of the app.
            </AlertDescription>
          </Alert>

          <div className="flex flex-col gap-3">
            <Button onClick={() => resend.mutate()} disabled={resend.isPending} className="w-full">
              {resend.isPending ? "Sending..." : "Resend verification email"}
            </Button>
            <Button variant="outline" onClick={() => void logout()} className="w-full">
              Sign out
            </Button>
          </div>
        </div>
      );
    }

    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Invalid verification link</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            This verification link is missing a token or is no longer usable.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Verify your email</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          We&apos;re confirming your email address now.
        </p>
      </div>

      {confirm.isPending && (
        <Alert>
          <AlertTitle>Checking your link</AlertTitle>
          <AlertDescription>This usually only takes a moment.</AlertDescription>
        </Alert>
      )}

      {confirm.isSuccess && (
        <Alert>
          <AlertTitle>Email verified</AlertTitle>
          <AlertDescription>Your email address has been confirmed successfully.</AlertDescription>
        </Alert>
      )}

      {confirm.isError && (
        <Alert variant="destructive">
          <AlertTitle>Verification failed</AlertTitle>
          <AlertDescription>
            {confirm.error instanceof Error
              ? confirm.error.message
              : "This link is invalid or expired."}
          </AlertDescription>
        </Alert>
      )}

      <div className="flex flex-col gap-3">
        {confirm.isSuccess && (
          <Button asChild className="w-full">
            <Link to={continueTo}>
              {isAuthenticated
                ? user?.onboarding_completed
                  ? "Go to overview"
                  : "Continue to onboarding"
                : "Go to sign in"}
            </Link>
          </Button>
        )}

        {confirm.isError && isAuthenticated && (
          <>
            <Button onClick={() => resend.mutate()} disabled={resend.isPending} className="w-full">
              {resend.isPending ? "Sending..." : "Send a new verification email"}
            </Button>
            <Button variant="outline" onClick={() => navigate(continueTo)} className="w-full">
              Back to app
            </Button>
          </>
        )}

        {confirm.isError && !isAuthenticated && (
          <Button asChild variant="outline" className="w-full">
            <Link to="/login">Back to sign in</Link>
          </Button>
        )}
      </div>
    </div>
  );
}
