import { PuzzleIcon } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs.tsx";
import { useUIStore } from "@/stores/ui-store.ts";
import { useCanDeleteProject } from "@/hooks/use-project-role.ts";
import { NotificationRecipientsSection } from "./components/notification-recipients-section.tsx";
import { AlertRulesSection } from "./components/alert-rules-section.tsx";
import { ProjectSection } from "./components/project-section.tsx";
import { MembersSection } from "./components/members-section.tsx";
import { DangerZoneSection } from "./components/danger-zone-section.tsx";

// ─── Tab nav styles ───────────────────────────────────────────────────────────

const tabListClass =
  "h-auto bg-transparent p-0 rounded-none w-full justify-start gap-1.5";

const tabTriggerClass =
  "rounded-lg px-3 py-1.5 text-sm font-medium " +
  "text-muted-foreground bg-transparent shadow-none " +
  "hover:text-foreground hover:bg-muted/30 transition-colors " +
  "data-[state=active]:text-foreground data-[state=active]:bg-muted/60";

// ─── Page ─────────────────────────────────────────────────────────────────────

export function SettingsPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const canDeleteProject = useCanDeleteProject(activeProjectId ?? 0);

  return (
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader title="Settings" />

      <Tabs defaultValue="notifications">
        <TabsList className={tabListClass}>
          <TabsTrigger value="notifications" className={tabTriggerClass}>Notifications</TabsTrigger>
          <TabsTrigger value="alert-rules" className={tabTriggerClass}>Alert Rules</TabsTrigger>
          <TabsTrigger value="project" className={tabTriggerClass}>Project</TabsTrigger>
          <TabsTrigger value="members" className={tabTriggerClass}>Members</TabsTrigger>
          <TabsTrigger value="integrations" className={tabTriggerClass}>Integrations</TabsTrigger>
          {canDeleteProject && (
            <TabsTrigger value="danger-zone" className={tabTriggerClass}>Danger Zone</TabsTrigger>
          )}
        </TabsList>

        {/* ── Notifications ── */}
        <TabsContent value="notifications" className="mt-6">
          {activeProjectId && <NotificationRecipientsSection projectId={activeProjectId} />}
        </TabsContent>

        {/* ── Alert Rules ── */}
        <TabsContent value="alert-rules" className="mt-6">
          {activeProjectId && <AlertRulesSection projectId={activeProjectId} />}
        </TabsContent>

        {/* ── Project ── */}
        <TabsContent value="project" className="mt-6">
          {activeProjectId && <ProjectSection projectId={activeProjectId} />}
        </TabsContent>

        {/* ── Members ── */}
        <TabsContent value="members" className="mt-6">
          {activeProjectId && <MembersSection projectId={activeProjectId} />}
        </TabsContent>

        {/* ── Integrations ── */}
        <TabsContent value="integrations" className="mt-6">
          <EmptyState
            icon={({ size, className }) => <PuzzleIcon size={size} className={className as string} />}
            title="Integrations coming soon"
            description="Integrations will be available in a future release."
          />
        </TabsContent>

        {/* ── Danger Zone ── */}
        {canDeleteProject && (
          <TabsContent value="danger-zone" className="mt-6">
            {activeProjectId && <DangerZoneSection projectId={activeProjectId} />}
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}
