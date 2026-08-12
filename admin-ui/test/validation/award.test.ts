import { describe, expect, it } from "vitest";
import { awardValidationSchema } from "@/lib/validation/award";

const mockT = ((key: string) => key) as TranslateFunction;
const schema = awardValidationSchema(mockT);

const valid: AwardFormValues = {
  title: "Best Game",
  issuer: "IGF",
  date: "2026-03-18",
  link: "",
  game_id: "",
};

describe("awardValidationSchema", () => {
  it("accepts a minimal award", async () => {
    await expect(schema.isValid(valid)).resolves.toBe(true);
  });

  it("requires a title and an issuer", async () => {
    await expect(schema.isValid({ ...valid, title: "" })).resolves.toBe(false);
    await expect(schema.isValid({ ...valid, title: "  " })).resolves.toBe(false);
    await expect(schema.isValid({ ...valid, issuer: "" })).resolves.toBe(false);
    await expect(schema.isValid({ ...valid, issuer: " " })).resolves.toBe(false);
  });

  describe("date", () => {
    it("requires a date", async () => {
      await expect(schema.isValid({ ...valid, date: "" })).resolves.toBe(false);
    });

    // The column is TEXT and the list is ordered by it as a string, so a
    // differently shaped date would sort into the wrong place.
    it("only accepts YYYY-MM-DD", async () => {
      for (const date of ["18/03/2026", "2026-3-1", "March 2026", "20260318"]) {
        await expect(schema.isValid({ ...valid, date })).resolves.toBe(false);
      }
      await expect(schema.isValid({ ...valid, date: "2026-03-18" })).resolves.toBe(
        true,
      );
    });
  });

  describe("link", () => {
    it("is optional", async () => {
      await expect(schema.isValid({ ...valid, link: "" })).resolves.toBe(true);
    });

    it("must be a full URL when given", async () => {
      await expect(
        schema.isValid({ ...valid, link: "example.com" }),
      ).resolves.toBe(false);
      await expect(
        schema.isValid({ ...valid, link: "https://example.com/a" }),
      ).resolves.toBe(true);
    });
  });

  // "No game" is the empty option in the picker, not a validation failure.
  it("treats an empty game as valid", async () => {
    await expect(schema.isValid({ ...valid, game_id: "" })).resolves.toBe(true);
  });
});
