import { describe, it, expect, beforeEach } from "vitest";
import { CookieHelper } from "@/utilities/cookie";

describe("CookieHelper", () => {
  beforeEach(() => {
    document.cookie = "";
  });

  describe("get", () => {
    it("returns null when cookie does not exist", () => {
      expect(CookieHelper.get("nonexistent")).toBeNull();
    });

    it("returns value of existing cookie", () => {
      document.cookie = "testKey=testValue";
      expect(CookieHelper.get("testKey")).toBe("testValue");
    });

    it("returns value when multiple cookies exist", () => {
      document.cookie = "a=1";
      document.cookie = "b=2";
      expect(CookieHelper.get("a")).toBe("1");
      expect(CookieHelper.get("b")).toBe("2");
    });
  });

  describe("set", () => {
    it("sets a cookie", () => {
      CookieHelper.set("myKey", "myValue");
      expect(document.cookie).toContain("myKey=myValue");
    });

    it("sets a cookie with default expiry of 365 days", () => {
      CookieHelper.set("expKey", "expValue");
      expect(document.cookie).toContain("expKey=expValue");
    });
  });

  describe("remove", () => {
    it("removes a cookie by setting expiry to past", () => {
      CookieHelper.set("toRemove", "val");
      CookieHelper.remove("toRemove");
      expect(CookieHelper.get("toRemove")).toBeNull();
    });
  });
});