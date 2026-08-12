import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import LanguageSwitcher from "@/components/layout/LanguageSwitcher";
import { useI18n } from "@/lib/stores/i18n";

describe("LanguageSwitcher", () => {
  beforeEach(() => {
    useI18n.getState().setLocale("en");
  });

  const openMenu = () => {
    fireEvent.click(screen.getAllByRole("button")[0]);
  };

  it("offers both languages", () => {
    render(<LanguageSwitcher />);
    openMenu();
    expect(screen.getByText(/English/)).toBeInTheDocument();
    expect(screen.getByText(/Türkçe/)).toBeInTheDocument();
  });

  // The panel previously had no switcher at all: the locale could only be
  // changed from the login screen, so it was unreachable once signed in.
  it("changes the locale when a language is picked", () => {
    render(<LanguageSwitcher />);
    openMenu();
    fireEvent.click(screen.getByText(/Türkçe/));

    expect(useI18n.getState().locale).toBe("tr");
  });

  it("translates through the store after switching", () => {
    render(<LanguageSwitcher />);
    openMenu();
    fireEvent.click(screen.getByText(/Türkçe/));

    expect(useI18n.getState().t("menu.games")).toBe("Oyunlar");
  });

  // The trigger shows a flag emoji, which a screen reader would announce as a
  // country rather than a language.
  it("gives the trigger an accessible name naming the language", () => {
    render(<LanguageSwitcher />);
    const trigger = screen.getAllByRole("button")[0];
    expect(trigger.getAttribute("aria-label")).toMatch(/English/);
  });
});
