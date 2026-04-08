import type { Severity } from "./alert.ts";

export type NotificationChannelType = "email";

export type NotificationRecipient = {
  id: number;
  project_id: number;
  channel_type: NotificationChannelType;
  target: string;
  min_severity: Severity;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type NotificationRecipientInput = {
  target: string;
  min_severity: Severity;
};

export type BulkSetRecipientsPayload = {
  channel_type: NotificationChannelType;
  project_id: number;
  recipients: NotificationRecipientInput[];
};

export type NotificationDelivery = {
  id: number;
  alert_id: number;
  recipient_id: number;
  channel: NotificationChannelType;
  target: string;
  delivered_at: string | null;
  failed_at: string | null;
  error_message: string | null;
};
