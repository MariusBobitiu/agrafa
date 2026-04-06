import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { authApi } from "@/data/auth.ts";
import { formatDate } from "@/lib/utils.ts";

export function SessionsSection() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["sessions"],
    queryFn: () => authApi.listSessions(),
  });

  const revoke = useMutation({
    mutationFn: (id: string) => authApi.deleteSession(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sessions"] }),
  });

  const revokeAll = useMutation({
    mutationFn: () => authApi.logoutAll(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["sessions"] });
      toast.success("All other sessions revoked");
    },
  });

  return (
    <div className="space-y-4 max-w-lg">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Active sessions</h2>
          <p className="text-sm text-muted-foreground">Manage your logged-in sessions.</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => revokeAll.mutate()}
          disabled={revokeAll.isPending}
        >
          Revoke all others
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}
        </div>
      ) : (
        data?.sessions?.map((session) => (
          <div
            key={session.id}
            className="flex items-center justify-between rounded-md border border-border px-3 py-2"
          >
            <div>
              <p className="text-sm font-medium">
                {session.is_current ? "Current session" : "Session"}
              </p>
              <p className="text-xs text-muted-foreground">
                Expires {formatDate(session.expires_at)}
              </p>
            </div>
            {!session.is_current && (
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive"
                onClick={() => revoke.mutate(session.id)}
                disabled={revoke.isPending}
              >
                Revoke
              </Button>
            )}
          </div>
        ))
      )}
    </div>
  );
}
