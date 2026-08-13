import { describe, expect, it } from "vitest";
import { parseExternalLinks } from "@/forms/GameForm";
import { gameValidationSchema } from "@/lib/validation/game";

const mockT = ((key: string) => key) as TranslateFunction;
const schema = gameValidationSchema(mockT);

const valid: GameFormValues = {
  title: "Pixel Quest",
  short_description: "",
  full_description: "",
  tags: "",
  genre: "",
  release_date: "",
  kind: "production",
  is_for_sale: false,
  price_display: "",
  external_links: [],
  is_published: false,
};

describe("gameValidationSchema", () => {
  it("accepts a minimal game", async () => {
    await expect(schema.isValid(valid)).resolves.toBe(true);
  });

  it("requires a title", async () => {
    await expect(schema.isValid({ ...valid, title: "" })).resolves.toBe(false);
  });

  it("rejects a whitespace-only title, matching the server's trim", async () => {
    await expect(schema.isValid({ ...valid, title: "   " })).resolves.toBe(
      false,
    );
  });

  describe("external links", () => {
    it("accepts a fully filled link", async () => {
      await expect(
        schema.isValid({
          ...valid,
          external_links: [{ label: "itch.io", url: "https://x.itch.io/g" }],
        }),
      ).resolves.toBe(true);
    });

    // Half a link renders as a broken link on the public site.
    it("rejects a link with no label", async () => {
      await expect(
        schema.isValid({
          ...valid,
          external_links: [{ label: "", url: "https://x.itch.io/g" }],
        }),
      ).resolves.toBe(false);
    });

    it("rejects a link with no url", async () => {
      await expect(
        schema.isValid({
          ...valid,
          external_links: [{ label: "itch.io", url: "" }],
        }),
      ).resolves.toBe(false);
    });

    it("rejects a url that is not a url", async () => {
      await expect(
        schema.isValid({
          ...valid,
          external_links: [{ label: "itch.io", url: "not a url" }],
        }),
      ).resolves.toBe(false);
    });
  });
});

describe("parseExternalLinks", () => {
  it("parses the JSON the API stores", () => {
    expect(
      parseExternalLinks('[{"label":"Itch","url":"https://a.dev"}]'),
    ).toEqual([{ label: "Itch", url: "https://a.dev" }]);
  });

  it("treats an empty value as no links", () => {
    expect(parseExternalLinks("")).toEqual([]);
    expect(parseExternalLinks("   ")).toEqual([]);
  });

  // Existing rows were hand-edited as raw JSON, so the edit screen has to
  // survive whatever is already in the column instead of crashing on it.
  it("degrades to an empty list on malformed JSON", () => {
    expect(parseExternalLinks("[{")).toEqual([]);
  });

  it("degrades to an empty list when the JSON is not an array", () => {
    expect(parseExternalLinks('{"label":"a"}')).toEqual([]);
  });

  it("fills in missing halves rather than dropping the row", () => {
    expect(
      parseExternalLinks('[{"label":"Itch"},{"url":"https://a.dev"}]'),
    ).toEqual([
      { label: "Itch", url: "" },
      { label: "", url: "https://a.dev" },
    ]);
  });

  it("skips entries that are not objects", () => {
    expect(parseExternalLinks('["nope", null, 3]')).toEqual([]);
  });
});

describe("release date and kind", () => {
  it("accepts a game with neither filled in", async () => {
    await expect(
      schema.isValid({ ...valid, release_date: "", kind: "production" }),
    ).resolves.toBe(true);
  });

  it("accepts a date the picker produces", async () => {
    await expect(
      schema.isValid({ ...valid, release_date: "2026-07-31" }),
    ).resolves.toBe(true);
  });

  // The API stores and sorts the date as text, so a date of another shape
  // would sort into the wrong place rather than fail loudly.
  it.each([
    "31-07-2026",
    "2026/07/31",
    "July 2026",
    "2026-7-31",
  ])("rejects %s", async (release_date) => {
    await expect(schema.isValid({ ...valid, release_date })).resolves.toBe(
      false,
    );
  });

  it("rejects a kind the public site has no badge for", async () => {
    await expect(
      schema.isValid({ ...valid, kind: "prototype" as GameKind }),
    ).resolves.toBe(false);
  });
});
