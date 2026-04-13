import { useNavigate } from "react-router-dom";
import { ArrowLeftIcon, HomeIcon, SearchXIcon } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { ErrorPageShell } from "@/components/error/error-page-shell.tsx";

export function NotFoundPage() {
  const navigate = useNavigate();

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
