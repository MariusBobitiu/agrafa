import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MonitorIcon } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import { Separator } from "@/components/ui/separator.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { authApi } from "@/data/auth.ts";
import { formatDate } from "@/lib/utils.ts";

// ─── Hook (reusable) ──────────────────────────────────────────────────────────

export function useSessionsData() {
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

  return { data, isLoading, revoke, revokeAll };
}

// ─── Header action button ─────────────────────────────────────────────────────

export function RevokeAllButton() {
  const { revokeAll } = useSessionsData();
  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => revokeAll.mutate()}
      disabled={revokeAll.isPending}
    >
      Revoke all others
    </Button>
  );
}

// ─── Session rows (headless — no own card/header) ─────────────────────────────

export function SessionsSection() {
  const { data, isLoading, revoke, revokeAll } = useSessionsData();

  if (isLoading) {
    return (
      <div className="divide-y divide-border">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="px-6 py-4">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="mt-1.5 h-3 w-28" />
          </div>
        ))}
      </div>
    );
  }

  const sessions = data?.sessions ?? [];

  return (
    <>
      {/* Card header */}
      <div className="flex items-center justify-between px-6 py-5">
        <div>
          <h2 className="text-sm font-semibold">Active sessions</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">Manage your logged-in sessions.</p>
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
      <Separator />
      {/* Rows */}
      <div className="divide-y divide-border">
        {sessions.map((session) => (
          <div key={session.id} className="flex items-center justify-between px-6 py-4">
            <div className="flex items-center gap-3">
              <MonitorIcon size={15} className="text-muted-foreground/50 shrink-0" />
              <div>
                <p className="text-sm font-medium">
                  {session.is_current ? "Current session" : "Session"}
                </p>
                <p className="text-xs text-muted-foreground">
                  Expires {formatDate(session.expires_at)}
                </p>
              </div>
            </div>
            {!session.is_current && (
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
                onClick={() => revoke.mutate(session.id)}
                disabled={revoke.isPending}
              >
                Revoke
              </Button>
            )}
          </div>
        ))}
      </div>
    </>
  );
}
