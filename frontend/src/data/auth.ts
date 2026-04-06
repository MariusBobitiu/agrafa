import { api } from "@/lib/fetch-client.ts";
import type {
  AuthSession,
  AuthSessionResponse,
  AuthStatusResponse,
  ForgotPasswordInput,
  LoginInput,
  RegisterInput,
  ResetPasswordInput,
  User,
  VerifyEmailConfirmInput,
  VerifyPasswordInput,
} from "@/types/auth.ts";

export const authApi = {
  me: (): Promise<{ user: User }> => api.get("/auth/me"),

  login: (input: LoginInput): Promise<AuthSessionResponse> =>
    api.post("/auth/login", input),

  register: (input: RegisterInput): Promise<AuthSessionResponse> =>
    api.post("/auth/register", input),

  logout: (): Promise<AuthStatusResponse> => api.post("/auth/logout"),

  completeOnboarding: (): Promise<{ user: User }> =>
    api.post("/auth/onboarding/complete"),

  sendVerifyEmail: (): Promise<AuthStatusResponse> =>
    api.post("/auth/verify-email/send"),

  confirmVerifyEmail: (input: VerifyEmailConfirmInput): Promise<AuthStatusResponse> =>
    api.post("/auth/verify-email/confirm", input),

  forgotPassword: (input: ForgotPasswordInput): Promise<AuthStatusResponse> =>
    api.post("/auth/forgot-password", input),

  resetPassword: (input: ResetPasswordInput): Promise<AuthStatusResponse> =>
    api.post("/auth/reset-password", input),

  verifyPassword: (input: VerifyPasswordInput): Promise<AuthStatusResponse> =>
    api.post("/auth/verify-password", input),

  listSessions: (): Promise<{ sessions: AuthSession[] }> =>
    api.get("/auth/sessions"),

  logoutAll: (): Promise<AuthStatusResponse> => api.post("/auth/logout-all"),

  deleteSession: (id: string): Promise<void> =>
    api.del(`/auth/sessions/${id}`),
};
