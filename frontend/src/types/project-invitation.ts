export type ProjectInvitationRole = "admin" | "viewer";

export type ProjectInvitation = {
  id: string;
  project_id: number;
  email: string;
  role: ProjectInvitationRole;
  invited_by_user_id: string;
  expires_at: string;
  accepted_at: string | null;
  created_at: string;
};

export type ProjectInvitationLookup = {
  id: string;
  project_id: number;
  project_name: string;
  email: string;
  role: ProjectInvitationRole;
  expires_at: string;
};

export type ProjectInvitationCreateItemInput = {
  email: string;
  role: ProjectInvitationRole;
};

export type ProjectInvitationCreateBatchInput = {
  project_id: number;
  invitations: ProjectInvitationCreateItemInput[];
};

export type ProjectInvitationCreateResult = {
  email: string;
  role: string;
  status: string;
  invitation?: ProjectInvitation;
  error_code?: string;
  error_message?: string;
};

export type ProjectInvitationCreateBatchResponse = {
  project_id: number;
  results: ProjectInvitationCreateResult[];
};

export type ProjectInvitationLookupResponse = {
  project_invitation: ProjectInvitationLookup;
};

export type ProjectInvitationAcceptInput = {
  token: string;
};

export type ProjectInvitationAcceptResponse = {
  status: string;
  already_member: boolean;
};
