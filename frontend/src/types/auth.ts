export type User = {
  id: string;
  name: string;
  email: string;
  email_verified: boolean;
  image: string | null;
  onboarding_completed: boolean;
  two_factor_enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type AuthSession = {
  id: string;
  expires_at: string;
  ip_address: string | null;
  user_agent: string | null;
  created_at: string;
  updated_at: string;
  is_current: boolean;
};

export type AuthSessionResponse = {
  user: User;
  expires_at: string;
};

export type AuthStatusResponse = {
  status: string;
};

export type LoginInput = {
  email: string;
  password: string;
  remember_me?: boolean;
};

export type RegisterInput = {
  name: string;
  email: string;
  password: string;
};

export type ForgotPasswordInput = {
  email: string;
};

export type ResetPasswordInput = {
  token: string;
  password: string;
};

export type VerifyEmailConfirmInput = {
  token: string;
};

export type VerifyPasswordInput = {
  password: string;
};
