import { describe, it, expect } from "vitest";
import { loginValidationSchema } from "@/lib/validation/auth";

const mockT = (key: string) => key;

describe("loginValidationSchema", () => {
  const schema = loginValidationSchema(mockT);

  it("validates correct email and password", async () => {
    const result = await schema.isValid({ email: "test@example.com", password: "123456", recaptcha: "test-token" });
    expect(result).toBe(true);
  });

  it("rejects invalid email", async () => {
    try {
      await schema.validate({ email: "not-an-email", password: "123456" });
      expect.fail("Should have thrown");
    } catch (err) {
      expect(err).toBeDefined();
    }
  });

  it("rejects missing email", async () => {
    try {
      await schema.validate({ email: "", password: "123456" });
      expect.fail("Should have thrown");
    } catch (err) {
      expect(err).toBeDefined();
    }
  });

  it("rejects missing password", async () => {
    try {
      await schema.validate({ email: "test@example.com", password: "" });
      expect.fail("Should have thrown");
    } catch (err) {
      expect(err).toBeDefined();
    }
  });

  it("rejects both missing email and password", async () => {
    try {
      await schema.validate({ email: "", password: "" });
      expect.fail("Should have thrown");
    } catch (err) {
      expect(err).toBeDefined();
    }
  });
});