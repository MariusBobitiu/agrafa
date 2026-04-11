import { api } from "@/lib/fetch-client.ts";
import type {
  BulkSetRecipientsPayload,
  NotificationDelivery,
  NotificationRecipient,
} from "@/types/notification.ts";

export const notificationsApi = {
  listRecipients: (projectId: number): Promise<{ notification_recipients: NotificationRecipient[] }> =>
    api.get(`/notification-recipients?project_id=${projectId}`),

  setRecipients: (
    payload: BulkSetRecipientsPayload,
  ): Promise<{ notification_recipients: NotificationRecipient[] }> =>
    api.post("/notification-recipients", payload),

  deleteRecipient: (id: number): Promise<void> =>
    api.del(`/notification-recipients/${id}`),

  listDeliveries: (projectId: number): Promise<{ deliveries: NotificationDelivery[] }> =>
    api.get(`/notification-deliveries?project_id=${projectId}`),

  sendTestEmail: (projectId: number, email: string): Promise<void> =>
    api.post("/notification-recipients/test-email", { project_id: projectId, email }),
};
