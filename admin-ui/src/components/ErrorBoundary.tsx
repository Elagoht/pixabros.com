import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "@/components/ui";
import { t } from "@/lib/stores/i18n";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
}

class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(_error: Error, _info: ErrorInfo): void {}

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <div className="flex h-screen w-full flex-col items-center justify-center gap-4">
            <p className="text-lg font-medium">
              {t("errorBoundary.somethingWentWrong")}
            </p>
            <Button
              variant="default"
              onClick={() => this.setState({ hasError: false })}
            >
              {t("errorBoundary.tryAgain")}
            </Button>
          </div>
        )
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
