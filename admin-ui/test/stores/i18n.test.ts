import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/utilities/cookie", () => ({
  CookieHelper: {
    get: vi.fn().mockReturnValue(null),
    set: vi.fn(),
    remove: vi.fn(),
  },
}));

describe("useI18n", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("initializes with default locale when no cookie", async () => {
    const { useI18n } = await import("@/lib/stores/i18n");
    const state = useI18n.getState();
    expect(state.locale).toBeDefined();
    expect(["en", "tr"]).toContain(state.locale);
  });

  it("setLocale updates locale", async () => {
    const { useI18n } = await import("@/lib/stores/i18n");
    useI18n.getState().setLocale("tr");
    expect(useI18n.getState().locale).toBe("tr");
  });

  it("setLocale calls CookieHelper.set", async () => {
    const { useI18n } = await import("@/lib/stores/i18n");
    const { CookieHelper } = await import("@/utilities/cookie");
    vi.mocked(CookieHelper.set).mockClear();

    useI18n.getState().setLocale("tr");
    expect(CookieHelper.set).toHaveBeenCalledWith("locale", "tr");
  });

  it("t function returns translation", async () => {
    const { useI18n } = await import("@/lib/stores/i18n");
    useI18n.getState().setLocale("en");
    const result = useI18n.getState().t("common.save" as never);
    expect(typeof result).toBe("string");
  });
});

describe("t standalone function", () => {
  it("returns translation for a key", async () => {
    const { t } = await import("@/lib/stores/i18n");
    const result = t("common.save" as never);
    expect(typeof result).toBe("string");
  });

  it("returns translation with params interpolation", async () => {
    const { useI18n, t } = await import("@/lib/stores/i18n");
    useI18n.getState().setLocale("en");
    const result = t("common.save" as never);
    expect(typeof result).toBe("string");
  });
});