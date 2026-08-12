import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ErrorBoundary from "@/components/ErrorBoundary";

const ThrowError = (): never => {
  throw new Error("Test error");
};

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders children normally when no error", () => {
    render(<ErrorBoundary><p>Hello world</p></ErrorBoundary>);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("catches errors and renders fallback", () => {
    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Something went wrong.")).toBeInTheDocument();
    expect(screen.getByText("Try again")).toBeInTheDocument();
  });

  it("renders custom fallback when provided", () => {
    render(
      <ErrorBoundary fallback={<p>Custom error UI</p>}>
        <ThrowError />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Custom error UI")).toBeInTheDocument();
  });

  it("resets state when Try again button is clicked", async () => {
    const user = userEvent.setup();

    let shouldThrow = true;

    const Child = () => {
      if (shouldThrow) throw new Error("boom");
      return <p>All good</p>;
    };

    render(
      <ErrorBoundary>
        <Child />
      </ErrorBoundary>,
    );

    expect(screen.getByText("Something went wrong.")).toBeInTheDocument();

    shouldThrow = false;
    await user.click(screen.getByText("Try again"));

    expect(screen.getByText("All good")).toBeInTheDocument();
  });
});