export type NotificationChannel = "email";

export type NotificationRecipient = {
  id: number;
  project_id: number;
  channel: NotificationChannel;
  target: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type NotificationDelivery = {
  id: number;
  alert_id: number;
  recipient_id: number;
  channel: NotificationChannel;
  target: string;
  delivered_at: string | null;
  failed_at: string | null;
  error_message: string | null;
};
