import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import GuestGuard from "@/components/guards/GuestGuard";

vi.mock("@/lib/stores/auth", () => ({
  useAuthStore: vi.fn(),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    Navigate: () => null,
    useSearchParams: () => [new URLSearchParams()],
  };
});

import { useAuthStore } from "@/lib/stores/auth";

const mockAuthStore = vi.mocked(useAuthStore);

const defaultMock = {
  isAuthenticated: false,
  isLoading: false,
  checkAuth: vi.fn(),
  setAuthenticated: vi.fn(),
  logout: vi.fn(),
};

const renderWithRouter = (children: React.ReactNode) =>
  render(<MemoryRouter>{children}</MemoryRouter>);

describe("GuestGuard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows Loading when isLoading", () => {
    mockAuthStore.mockReturnValue({ ...defaultMock, isLoading: true });
    renderWithRouter(<GuestGuard>Login Page</GuestGuard>);
    expect(document.querySelector(".animate-spin")).toBeInTheDocument();
  });

  it("redirects when authenticated", () => {
    mockAuthStore.mockReturnValue({
      ...defaultMock,
      isAuthenticated: true,
      isLoading: false,
    });
    const { container } = renderWithRouter(<GuestGuard>Login Page</GuestGuard>);
    expect(container.textContent).not.toContain("Login Page");
  });

  it("renders children when not authenticated", () => {
    mockAuthStore.mockReturnValue({
      ...defaultMock,
      isAuthenticated: false,
      isLoading: false,
    });
    renderWithRouter(<GuestGuard>Login Page</GuestGuard>);
    expect(screen.getByText("Login Page")).toBeInTheDocument();
  });
});
