import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { authApi } from "@/data/auth.ts";
import { projectInvitationsApi } from "@/data/project-invitations.ts";
import { useAuth } from "@/hooks/use-auth.ts";
import { buildRedirectPath, setAuthRedirect } from "@/lib/auth-redirect.ts";
import { formatDate } from "@/lib/utils.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { useMeta } from "@/hooks/use-meta";

export function InvitePage() {
  useMeta({
    title: "Project Invitation",
    description: "Review and accept your project invitation to get started with Agrafa",
  });
  const { isAuthenticated, user, logout, refreshUser } = useAuth();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token")?.trim() ?? "";
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const setActiveProjectId = useUIStore((state) => state.setActiveProjectId);

  const inviteQuery = useQuery({
    queryKey: ["project-invitations", "by-token", token],
    queryFn: () => projectInvitationsApi.getByToken(token),
    enabled: token.length > 0,
    retry: false,
  });

  const invite = inviteQuery.data?.project_invitation;
  const emailMatches =
    user && invite ? user.email.trim().toLowerCase() === invite.email.trim().toLowerCase() : true;

  const accept = useMutation({
    mutationFn: async () => {
      const result = await projectInvitationsApi.accept({ token });

      if (invite) {
        setActiveProjectId(invite.project_id);
      }

      await queryClient.invalidateQueries({ queryKey: ["projects"] });
      await queryClient.refetchQueries({ queryKey: ["projects"], type: "active" });

      if (user && !user.onboarding_completed) {
        await authApi.completeOnboarding();
      }

      await refreshUser();

      return result;
    },
    onSuccess: (result) => {
      toast.success(
        result.already_member ? "Project already linked to your account." : "Invitation accepted.",
      );
      navigate("/overview", { replace: true });
    },
  });

  function handleAuthRedirect(path: "/sign-in" | "/sign-up") {
    setAuthRedirect(buildRedirectPath(location));
    navigate(path, { replace: true });
  }

  async function handleSwitchAccount() {
    const redirectPath = buildRedirectPath(location);
    await logout();
    setAuthRedirect(redirectPath);
  }

  if (!token) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Invalid invitation</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            This invitation link is missing a token or is no longer valid.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Project invitation</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Review the invite details and accept it with the matching account.
        </p>
      </div>

      {inviteQuery.isLoading && (
        <Alert>
          <AlertTitle>Loading invitation</AlertTitle>
          <AlertDescription>Please wait while we verify the invite token.</AlertDescription>
        </Alert>
      )}

      {inviteQuery.isError && (
        <Alert variant="destructive">
          <AlertTitle>Invitation unavailable</AlertTitle>
          <AlertDescription>
            {inviteQuery.error instanceof Error
              ? inviteQuery.error.message
              : "This invitation is invalid or expired."}
          </AlertDescription>
        </Alert>
      )}

      {invite && (
        <>
          <div className="space-y-3 rounded-lg border border-border p-5">
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-sm text-muted-foreground">Project</p>
                <h2 className="text-lg font-semibold">
                  {invite.project_name || "Untitled project"}
                </h2>
              </div>
              <Badge className="capitalize">{invite.role}</Badge>
            </div>

            <div className="grid gap-12 sm:grid-cols-2">
              <div>
                <p className="text-sm text-muted-foreground">Invited email</p>
                <p className="font-medium">{invite.email}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Expires</p>
                <p className="font-medium">{formatDate(invite.expires_at)}</p>
              </div>
            </div>
          </div>

          {!isAuthenticated ? (
            <Alert>
              <AlertTitle>Sign in to accept</AlertTitle>
              <AlertDescription>
                Use the invited email address, and we&apos;ll bring you back to this invitation
                after authentication.
              </AlertDescription>
            </Alert>
          ) : !emailMatches ? (
            <Alert variant="destructive">
              <AlertTitle>Wrong account</AlertTitle>
              <AlertDescription>
                You are signed in as {user?.email}, but this invitation was sent to {invite.email}.
              </AlertDescription>
            </Alert>
          ) : null}

          {accept.isError && (
            <Alert variant="destructive">
              <AlertTitle>Could not accept invitation</AlertTitle>
              <AlertDescription>
                {accept.error instanceof Error ? accept.error.message : "Please try again."}
              </AlertDescription>
            </Alert>
          )}

          <div className="flex flex-col gap-3">
            {!isAuthenticated ? (
              <>
                <Button onClick={() => handleAuthRedirect("/sign-in")} className="w-full">
                  Sign in to accept
                </Button>
                <Button
                  variant="outline"
                  onClick={() => handleAuthRedirect("/sign-up")}
                  className="w-full"
                >
                  Create account
                </Button>
              </>
            ) : (
              <>
                <Button
                  onClick={() => accept.mutate()}
                  disabled={!emailMatches || accept.isPending}
                  className="w-full"
                >
                  {accept.isPending ? "Accepting..." : "Accept invitation"}
                </Button>
                {!emailMatches && (
                  <Button
                    variant="outline"
                    onClick={() => void handleSwitchAccount()}
                    className="w-full"
                  >
                    Sign out
                  </Button>
                )}
              </>
            )}
          </div>
        </>
      )}
    </div>
  );
}
