import { PageHeader } from "@/components/ui/page-header.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs.tsx";
import { useUIStore } from "@/stores/ui-store.ts";
import { NotificationRecipientsSection } from "./components/notification-recipients-section.tsx";
import { SessionsSection } from "./components/sessions-section.tsx";
import { GeneralSection } from "./components/general-section.tsx";

export function SettingsPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);

  return (
    <div className="p-6 space-y-6">
      <PageHeader title="Settings" />

      <Tabs defaultValue="general">
        <TabsList>
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
          <TabsTrigger value="sessions">Sessions</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="mt-4">
          {activeProjectId && <GeneralSection projectId={activeProjectId} />}
        </TabsContent>

        <TabsContent value="notifications" className="mt-4">
          {activeProjectId && <NotificationRecipientsSection projectId={activeProjectId} />}
        </TabsContent>

        <TabsContent value="sessions" className="mt-4">
          <SessionsSection />
        </TabsContent>
      </Tabs>
    </div>
  );
}
