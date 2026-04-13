import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangleIcon, RefreshCwIcon } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { ErrorPageShell } from "./error-page-shell.tsx";

type Props = { children: ReactNode };
type State = { error: Error | null };

export class GlobalErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("[GlobalErrorBoundary]", error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ error: null });
  };

  override render() {
    if (this.state.error) {
      return (
        <div className="bg-background h-screen">
          <ErrorPageShell
            ghost="Oops"
            icon={<AlertTriangleIcon size={28} className="text-destructive" />}
            title="Something went wrong"
            description="An unexpected error occurred. Try reloading the page."
            actions={
              <>
                <Button variant="outline" size="sm" onClick={this.handleReset}>
                  Try again
                </Button>
                <Button variant="default" size="sm" onClick={() => window.location.reload()}>
                  <RefreshCwIcon size={14} />
                  Reload
                </Button>
              </>
            }
          />
        </div>
      );
    }

    return this.props.children;
  }
}
