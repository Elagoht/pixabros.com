import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

vi.mock("@/lib/stores/auth", () => ({
  useAuthStore: vi.fn(() => ({
    isAuthenticated: false,
    isLoading: false,
    checkAuth: vi.fn(),
    setAuthenticated: vi.fn(),
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
  sessionService: { create: vi.fn() },
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
  it("renders email and password fields", () => {
    renderForm();
    expect(screen.getByPlaceholderText("auth.email *")).toBeInTheDocument();
    expect(document.querySelector('input[name="password"]')).toBeTruthy();
  });

  it("renders submit button", () => {
    renderForm();
    const submitButton = screen.getByRole("button", { name: "auth.login" });
    expect(submitButton).toBeInTheDocument();
  });
});