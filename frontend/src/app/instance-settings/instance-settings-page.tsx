import { PuzzleIcon } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs.tsx";
import { InstanceSection } from "@/app/settings/components/instance-section.tsx";
import { useMeta } from "@/hooks/use-meta.ts";

// ─── Tab nav styles ───────────────────────────────────────────────────────────

const tabListClass = "h-auto bg-transparent p-0 rounded-none w-full justify-start gap-1.5";

const tabTriggerClass =
  "rounded-lg px-3 py-1.5 text-sm font-medium " +
  "text-muted-foreground bg-transparent shadow-none " +
  "hover:text-foreground hover:bg-muted/30 transition-colors " +
  "data-[state=active]:text-foreground data-[state=active]:bg-muted/60";

// ─── Page ─────────────────────────────────────────────────────────────────────

export function InstanceSettingsPage() {
  useMeta({
    title: "Instance Settings",
    description: "Global deployment configuration for this Agrafa instance",
  });

  return (
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader
        title="Instance Settings"
        description="Global configuration for this self-hosted deployment"
      />

      <Tabs defaultValue="instance">
        <TabsList className={tabListClass}>
          <TabsTrigger value="instance" className={tabTriggerClass}>
            Instance
          </TabsTrigger>
          <TabsTrigger value="integrations" className={tabTriggerClass}>
            Integrations
          </TabsTrigger>
        </TabsList>

        {/* ── Instance ── */}
        <TabsContent value="instance" className="mt-6">
          <InstanceSection />
        </TabsContent>

        {/* ── Integrations ── */}
        <TabsContent value="integrations" className="mt-6">
          <EmptyState
            icon={({ size, className }) => (
              <PuzzleIcon size={size} className={className as string} />
            )}
            title="Integrations coming soon"
            description="Webhook and third-party integrations will be available in a future release."
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
