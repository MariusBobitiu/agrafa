import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ChevronRightIcon, GlobeIcon, NetworkIcon } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { useServices } from "@/hooks/use-services.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import { cn } from "@/lib/utils.ts";
import { useServiceCreationStore } from "@/stores/service-creation-store.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { CreateNodeDialog } from "../nodes/components/create-node-dialog.tsx";
import { CreateServiceDialog } from "./components/create-service-dialog.tsx";
import { AnimateIcon, PlusIcon, ActivityIcon } from "@/components/animate-ui/icons";
import type { Service } from "@/types/service.ts";

// ─── Service row card ─────────────────────────────────────────────────────────

function ServiceRowCard({ svc }: { svc: Service }) {
  const navigate = useNavigate();
  const latencyMs = svc.latest_health_check?.response_time_ms ?? null;
  const isUnhealthy = svc.status === "unhealthy";

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => navigate(`/services/${svc.id}`)}
      onKeyDown={(e) => e.key === "Enter" && navigate(`/services/${svc.id}`)}
			className={cn(
        "group flex items-center gap-4 rounded-xl border px-5 py-3.5 transition-colors cursor-pointer",
				"hover:bg-muted/30",
				isUnhealthy
					? "border-destructive/25 bg-destructive/2"
					: "border-border bg-card",
			)}
		>
			{/* ── Left: identity ── */}
			<div className="flex items-center gap-3 w-64 shrink-0 min-w-0">
				<div
					className={cn(
						"flex size-9 items-center justify-center rounded-lg shrink-0",
						svc.status === "healthy"
							? "bg-primary/10"
							: svc.status === "unhealthy"
								? "bg-destructive/10"
								: svc.status === "degraded"
									? "bg-warning/10"
									: "bg-muted",
					)}
				>
					<GlobeIcon
						size={16}
						className={cn(
							svc.status === "healthy"
								? "text-primary"
								: svc.status === "unhealthy"
									? "text-destructive"
									: svc.status === "degraded"
										? "text-warning"
										: "text-muted-foreground",
						)}
					/>
				</div>
				<div className="min-w-0">
					<p className="text-sm font-semibold text-foreground truncate leading-snug">
						{svc.name}
					</p>
					<div className="flex items-center gap-1 mt-0.5">
						{svc.check_type === "http" ? (
							<GlobeIcon
								size={10}
								className="text-muted-foreground/40 shrink-0"
							/>
						) : (
							<NetworkIcon
								size={10}
								className="text-muted-foreground/40 shrink-0"
							/>
						)}
						<p className="text-xs text-muted-foreground/70 font-mono truncate">
							{svc.check_target}
						</p>
					</div>
				</div>
			</div>

			{/* ── Spacer ── */}
			<div className="flex-1 min-w-0" />

      {/* ── Right: primary row + secondary recency + chevron ── */}
      <div className="flex items-center gap-4 shrink-0">
        <div className="flex flex-col items-end gap-1">
          <div className="flex items-center justify-end gap-3">
            <p
              className={cn(
                "text-sm font-medium tabular-nums leading-none",
                latencyMs === null ? "text-muted-foreground/30" : "text-foreground",
              )}
            >
              {latencyMs !== null ? `${latencyMs}ms` : "—"}
            </p>
            <StatusBadge status={svc.status} />
          </div>
          <p className="text-xs text-muted-foreground/60 leading-none">
            {svc.last_checked_at
              ? `Checked ${formatRelativeTime(svc.last_checked_at)}`
              : "Never checked"}
          </p>
        </div>
				<ChevronRightIcon
					size={13}
					className="text-muted-foreground/20 group-hover:text-muted-foreground/60 transition-colors ml-1"
				/>
			</div>
		</div>
	);
}

// ─── Skeleton row ─────────────────────────────────────────────────────────────

function ServiceRowSkeleton() {
  return (
    <div className="flex items-center gap-4 rounded-xl border border-border bg-card px-5 py-4">
      <div className="flex items-center gap-3 w-64 shrink-0">
        <Skeleton className="h-2 w-2 rounded-full" />
        <div className="space-y-1.5">
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-3 w-24" />
        </div>
      </div>
      <div className="flex-1" />
      <div className="flex items-center gap-4 shrink-0">
        <div className="flex flex-col items-end gap-1.5">
          <div className="flex items-center gap-3">
            <Skeleton className="h-3.5 w-14" />
            <Skeleton className="h-5 w-16 rounded-full" />
          </div>
          <Skeleton className="h-3 w-20" />
        </div>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function ServicesPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const markServiceDraftResume = useServiceCreationStore((s) => s.markResume);
  const { data, isLoading, error } = useServices(activeProjectId ?? 0);
  const [searchParams, setSearchParams] = useSearchParams();
  const createOpen = searchParams.get("create") === "1";
  const [nodeSetupOpen, setNodeSetupOpen] = useState(false);
  const [, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => {
      window.clearInterval(timer);
    };
  }, []);

  function setCreateOpen(open: boolean) {
    const nextParams = new URLSearchParams(searchParams);
    if (open) {
      nextParams.set("create", "1");
    } else {
      nextParams.delete("create");
    }

    setSearchParams(nextParams, { replace: true });
  }

  function handleRequestNodeSetup() {
    setCreateOpen(false);
    setNodeSetupOpen(true);
  }

  function handleNodeSetupOpenChange(open: boolean) {
    setNodeSetupOpen(open);
    if (!open) {
      markServiceDraftResume();
      setCreateOpen(true);
    }
  }

  function handleNodeSetupComplete(nodeId: number) {
    markServiceDraftResume(nodeId.toString());
    setNodeSetupOpen(false);
    setCreateOpen(true);
  }

  return (
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader
        title="Services"
        description="HTTP and TCP service health checks"
        actions={
          <AnimateIcon asChild animateOnHover>
            <Button size="sm" onClick={() => setCreateOpen(true)} disabled={!activeProjectId}>
              <PlusIcon size={14} className="mr-1.5" />
              Add service
            </Button>
          </AnimateIcon>
        }
      />

      {error && <p className="text-sm text-destructive">Failed to load services.</p>}

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <ServiceRowSkeleton key={i} />
          ))}
        </div>
      ) : !data?.services?.length ? (
        <EmptyState
          icon={ActivityIcon}
          title="No services yet"
          description="Monitor a service from this instance or your own server."
          action={{ label: "Monitor a service", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="space-y-2">
          {data.services.map((svc) => (
            <ServiceRowCard key={svc.id} svc={svc} />
          ))}
        </div>
      )}

      {activeProjectId && (
        <>
          <CreateServiceDialog
            projectId={activeProjectId}
            open={createOpen}
            onOpenChange={setCreateOpen}
            onRequestNodeSetup={handleRequestNodeSetup}
          />
          <CreateNodeDialog
            projectId={activeProjectId}
            open={nodeSetupOpen}
            onOpenChange={handleNodeSetupOpenChange}
            launchedFromService
            onComplete={handleNodeSetupComplete}
          />
        </>
      )}
    </div>
  );
}
