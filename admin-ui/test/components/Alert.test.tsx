import { describe, it, expect } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { Alert } from "@/components/ui";
import { IconInfoCircle } from "@tabler/icons-react";

describe("Alert", () => {
  it("renders title and description", () => {
    render(<Alert title="Info" description="Some details" />);
    expect(screen.getByText("Info")).toBeInTheDocument();
    expect(screen.getByText("Some details")).toBeInTheDocument();
  });

  it("renders title without description", () => {
    render(<Alert title="Warning" />);
    expect(screen.getByText("Warning")).toBeInTheDocument();
  });

  it("applies info variant classes by default", () => {
    const { container } = render(<Alert title="Info" />);
    expect(container.querySelector(".border-blue-400")).toBeTruthy();
  });

  it("applies success variant classes", () => {
    const { container } = render(<Alert variant="success" title="Success" />);
    expect(container.querySelector(".border-green-400")).toBeTruthy();
  });

  it("applies warning variant classes", () => {
    const { container } = render(<Alert variant="warning" title="Warning" />);
    expect(container.querySelector(".border-yellow-400")).toBeTruthy();
  });

  it("applies error variant classes", () => {
    const { container } = render(<Alert variant="error" title="Error" />);
    expect(container.querySelector(".border-red-400")).toBeTruthy();
  });

  it("renders custom icon", () => {
    render(<Alert title="Custom" icon={IconInfoCircle} />);
    expect(screen.getByText("Custom")).toBeInTheDocument();
  });

  it("hides alert when closable and close button clicked", async () => {
    render(<Alert title="Closable" closable />);
    const closeBtn = screen.getByRole("button");
    await act(() => { closeBtn.click(); });
    expect(screen.queryByText("Closable")).not.toBeInTheDocument();
  });
});