import { AlertTriangle } from "lucide-react";
import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "@/components/ui/Button";

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

// Catches render-time errors anywhere below it so a bug in one page can't
// blank the whole dashboard; the rest of the app (sidebar, nav) stays
// usable and the user gets a real message instead of a white screen.
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Unhandled error in PulseWatch UI:", error, info.componentStack);
  }

  render() {
    if (!this.state.error) {
      return this.props.children;
    }

    return (
      <div className="flex flex-col items-center justify-center gap-3 p-16 text-center">
        <AlertTriangle className="size-8 text-down" />
        <p className="text-sm font-medium text-fg">Something went wrong</p>
        <p className="max-w-sm text-sm text-fg-muted">{this.state.error.message}</p>
        <Button variant="secondary" onClick={() => this.setState({ error: null })}>
          Try again
        </Button>
      </div>
    );
  }
}
