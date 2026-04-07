import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner.tsx";
import { AuthProvider } from "@/contexts/auth-context.tsx";
import { RequireAuth } from "@/components/auth/require-auth.tsx";
import { RequireGuest } from "@/components/auth/require-guest.tsx";
import { AppLayout } from "@/components/layout/app-layout.tsx";
import { AuthLayout } from "@/components/layout/auth-layout.tsx";
import { LoginPage } from "@/app/auth/login-page.tsx";
import { RegisterPage } from "@/app/auth/register-page.tsx";
import { ForgotPasswordPage } from "@/app/auth/forgot-password-page.tsx";
import { ResetPasswordPage } from "@/app/auth/reset-password-page.tsx";
import { VerifyEmailPage } from "@/app/auth/verify-email-page.tsx";
import { OnboardingPage } from "@/app/onboarding/onboarding-page.tsx";
import { OverviewPage } from "@/app/overview/overview-page.tsx";
import { NodesPage } from "@/app/nodes/nodes-page.tsx";
import { NodeDetailPage } from "@/app/nodes/node-detail-page.tsx";
import { ServicesPage } from "@/app/services/services-page.tsx";
import { AlertsPage } from "@/app/alerts/alerts-page.tsx";
import { SettingsPage } from "@/app/settings/settings-page.tsx";
import { InvitePage } from "@/app/invite/invite-page.tsx";

// Root layout — wraps everything in AuthProvider (which needs useNavigate from the router)
function RootLayout() {
  return (
    <AuthProvider>
      <Outlet />
      <Toaster />
    </AuthProvider>
  );
}

export const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      {
        element: <AuthLayout />,
        children: [
          { path: "/forgot-password", element: <ForgotPasswordPage /> },
          { path: "/reset-password", element: <ResetPasswordPage /> },
          { path: "/verify-email", element: <VerifyEmailPage /> },
          { path: "/invite", element: <InvitePage /> },
        ],
      },

      // Public routes (redirect to /overview if already authenticated)
      {
        element: <RequireGuest />,
        children: [
          {
            element: <AuthLayout />,
            children: [
              { path: "/login", element: <LoginPage /> },
              { path: "/register", element: <RegisterPage /> },
            ],
          },
        ],
      },

      // Protected routes — no /app prefix in URL
      {
        element: <RequireAuth />,
        children: [
          { path: "/onboarding", element: <OnboardingPage /> },
          {
            element: <AppLayout />,
            children: [
              { path: "/overview", element: <OverviewPage /> },
              { path: "/nodes", element: <NodesPage /> },
              { path: "/nodes/:id", element: <NodeDetailPage /> },
              { path: "/services", element: <ServicesPage /> },
              { path: "/alerts", element: <AlertsPage /> },
              { path: "/settings", element: <SettingsPage /> },
            ],
          },
        ],
      },

      // Catch-all
      { path: "*", element: <Navigate to="/overview" replace /> },
    ],
  },
]);
