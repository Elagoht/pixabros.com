import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/stores/auth", () => ({
  useAuthStore: vi.fn(() => ({
    isAuthenticated: false,
    isLoading: false,
    checkAuth: vi.fn(),
    setAuthenticated: vi.fn(),
    setSession: vi.fn(),
    logout: vi.fn(),
  })),
}));

vi.mock("@/lib/stores/i18n", () => ({
  useI18n: vi.fn(() => ({
    t: (key: string) => key,
    locale: "en",
    setLocale: vi.fn(),
  })),
}));

vi.mock("@/hooks/useLoginRedirect", () => ({
  useLoginRedirect: vi.fn(() => vi.fn()),
}));

vi.mock("@/services/session", () => ({
  SessionService: { create: vi.fn() },
}));

vi.mock("@/lib/validation/auth", () => ({
  loginValidationSchema: () => ({
    validate: vi.fn(),
  }),
}));

import LoginForm from "@/forms/LoginForm";

const renderForm = () =>
  render(
    <MemoryRouter>
      <LoginForm />
    </MemoryRouter>,
  );

describe("LoginForm", () => {
  it("renders a username field, not an email field", () => {
    renderForm();
    expect(screen.getByPlaceholderText("auth.username *")).toBeInTheDocument();
    expect(document.querySelector('input[name="username"]')).toBeTruthy();
    expect(document.querySelector('input[name="email"]')).toBeNull();
  });

  it("renders a password field", () => {
    renderForm();
    expect(document.querySelector('input[name="password"]')).toBeTruthy();
  });

  it("renders submit button", () => {
    renderForm();
    expect(
      screen.getByRole("button", { name: "auth.login" }),
    ).toBeInTheDocument();
  });

  it("links to the forgot-password page", () => {
    renderForm();
    expect(
      screen.getByRole("link", { name: "auth.forgotPassword" }),
    ).toHaveAttribute("href", "/forgot-password");
  });
});
