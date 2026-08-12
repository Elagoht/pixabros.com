import { describe, expect, it } from "vitest";
import { devlogValidationSchema } from "@/lib/validation/devlog";

const mockT = ((key: string) => key) as TranslateFunction;
const schema = devlogValidationSchema(mockT);

const valid: DevlogFormValues = {
  title: "Kartuş sistemi geldi",
  content_markdown: "",
  game_id: "",
  is_published: false,
  published_at: "",
};

describe("devlogValidationSchema", () => {
  it("accepts a minimal draft", async () => {
    await expect(schema.isValid(valid)).resolves.toBe(true);
  });

  it("requires a title", async () => {
    await expect(schema.isValid({ ...valid, title: "" })).resolves.toBe(false);
    await expect(schema.isValid({ ...valid, title: "   " })).resolves.toBe(
      false,
    );
  });

  // The server stamps a date on first publish, so an empty one is normal.
  it("allows an empty publication date", async () => {
    await expect(
      schema.isValid({ ...valid, published_at: "" }),
    ).resolves.toBe(true);
  });

  it("only accepts YYYY-MM-DD when a date is given", async () => {
    for (const date of ["12/08/2026", "2026-8-1", "yesterday"]) {
      await expect(
        schema.isValid({ ...valid, published_at: date }),
      ).resolves.toBe(false);
    }
    await expect(
      schema.isValid({ ...valid, published_at: "2026-08-12" }),
    ).resolves.toBe(true);
  });
});
