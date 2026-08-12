import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import AuthGuard from "@/components/guards/AuthGuard";

vi.mock("@/lib/stores/auth", () => ({
  useAuthStore: vi.fn(),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    Navigate: () => null,
    useLocation: () => ({ pathname: "/dashboard", search: "", hash: "" }),
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

describe("AuthGuard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows Loading when isLoading", () => {
    mockAuthStore.mockReturnValue({ ...defaultMock, isLoading: true });
    renderWithRouter(<AuthGuard>Dashboard</AuthGuard>);
    expect(document.querySelector(".animate-spin")).toBeInTheDocument();
  });

  it("redirects to login when not authenticated", () => {
    mockAuthStore.mockReturnValue({ ...defaultMock, isAuthenticated: false, isLoading: false });
    const { container } = renderWithRouter(<AuthGuard>Dashboard</AuthGuard>);
    expect(container.textContent).not.toContain("Dashboard");
  });

  it("renders children when authenticated", () => {
    mockAuthStore.mockReturnValue({ ...defaultMock, isAuthenticated: true, isLoading: false });
    renderWithRouter(<AuthGuard>Dashboard</AuthGuard>);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });
});