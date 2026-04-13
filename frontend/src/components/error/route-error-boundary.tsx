import { useRouteError, isRouteErrorResponse, useNavigate } from "react-router-dom";
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  HomeIcon,
  RefreshCwIcon,
  SearchXIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { ErrorPageShell } from "./error-page-shell.tsx";

export function RouteErrorBoundary() {
  const error = useRouteError();
  const navigate = useNavigate();

  if (isRouteErrorResponse(error) && error.status === 404) {
    return (
      <ErrorPageShell
        ghost="404"
        icon={<SearchXIcon size={28} className="text-muted-foreground" />}
        title="Page not found"
        description="The page you're looking for doesn't exist or may have been moved."
        actions={
          <>
            <Button variant="outline" size="sm" onClick={() => navigate(-1)}>
              <ArrowLeftIcon size={14} />
              Go back
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={() => navigate("/overview", { replace: true })}
            >
              <HomeIcon size={14} />
              Dashboard
            </Button>
          </>
        }
      />
    );
  }

  console.error("[RouteErrorBoundary]", error);

  return (
    <ErrorPageShell
      ghost="Oops"
      icon={<AlertTriangleIcon size={28} className="text-destructive" />}
      title="Something went wrong"
      description="An unexpected error occurred. Try reloading the page."
      actions={
        <>
          <Button variant="outline" size="sm" onClick={() => navigate(-1)}>
            <ArrowLeftIcon size={14} />
            Go back
          </Button>
          <Button variant="outline" size="sm" onClick={() => window.location.reload()}>
            <RefreshCwIcon size={14} />
            Reload
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={() => navigate("/overview", { replace: true })}
          >
            <HomeIcon size={14} />
            Dashboard
          </Button>
        </>
      }
    />
  );
}
