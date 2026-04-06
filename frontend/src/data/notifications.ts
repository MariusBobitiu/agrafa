import { api } from "@/lib/fetch-client.ts";
import type { NotificationDelivery, NotificationRecipient } from "@/types/notification.ts";

export const notificationsApi = {
  listRecipients: (projectId: number): Promise<{ recipients: NotificationRecipient[] }> =>
    api.get(`/notification-recipients?project_id=${projectId}`),

  createRecipient: (
    payload: Pick<NotificationRecipient, "project_id" | "channel" | "target">,
  ): Promise<{ recipient: NotificationRecipient }> =>
    api.post("/notification-recipients", payload),

  updateRecipient: (
    id: number,
    payload: Partial<Pick<NotificationRecipient, "is_enabled">>,
  ): Promise<{ recipient: NotificationRecipient }> =>
    api.patch(`/notification-recipients/${id}`, payload),

  deleteRecipient: (id: number): Promise<void> =>
    api.del(`/notification-recipients/${id}`),

  listDeliveries: (projectId: number): Promise<{ deliveries: NotificationDelivery[] }> =>
    api.get(`/notification-deliveries?project_id=${projectId}`),
};
