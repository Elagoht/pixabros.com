import { describe, expect, it } from "vitest";
import {
  changePasswordValidationSchema,
  loginValidationSchema,
} from "@/lib/validation/auth";

const mockT = ((key: string) => key) as TranslateFunction;

describe("loginValidationSchema", () => {
  const schema = loginValidationSchema(mockT);

  it("accepts a username and password", async () => {
    await expect(
      schema.isValid({ username: "furkan", password: "s3cret-password" }),
    ).resolves.toBe(true);
  });

  it("rejects a missing username", async () => {
    await expect(
      schema.isValid({ username: "", password: "s3cret-password" }),
    ).resolves.toBe(false);
  });

  it("rejects a missing password", async () => {
    await expect(
      schema.isValid({ username: "furkan", password: "" }),
    ).resolves.toBe(false);
  });
});

describe("changePasswordValidationSchema", () => {
  const schema = changePasswordValidationSchema(mockT);

  const valid = {
    current_password: "old-password",
    new_password: "new-password",
    confirm_password: "new-password",
  };

  it("accepts a well-formed change", async () => {
    await expect(schema.isValid(valid)).resolves.toBe(true);
  });

  it("rejects a mismatched confirmation", async () => {
    await expect(
      schema.isValid({ ...valid, confirm_password: "something-else" }),
    ).resolves.toBe(false);
  });

  // Mirrors auth.ValidatePassword on the Go side.
  it("rejects a new password under 8 characters", async () => {
    await expect(
      schema.isValid({
        ...valid,
        new_password: "short",
        confirm_password: "short",
      }),
    ).resolves.toBe(false);
  });

  it("rejects a new password over 72 characters", async () => {
    const tooLong = "a".repeat(73);
    await expect(
      schema.isValid({
        ...valid,
        new_password: tooLong,
        confirm_password: tooLong,
      }),
    ).resolves.toBe(false);
  });

  it("accepts a new password of exactly 72 characters", async () => {
    const atLimit = "a".repeat(72);
    await expect(
      schema.isValid({
        ...valid,
        new_password: atLimit,
        confirm_password: atLimit,
      }),
    ).resolves.toBe(true);
  });
});
