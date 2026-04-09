export type ProjectMemberRole = "owner" | "admin" | "viewer";

export type ProjectMemberUser = {
  name: string;
  email: string;
  image: string | null;
};

export type ProjectMember = {
  id: string;
  project_id: number;
  user_id: string;
  role: ProjectMemberRole;
  created_at: string;
  user: ProjectMemberUser;
};
