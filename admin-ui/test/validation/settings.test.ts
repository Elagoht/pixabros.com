import { describe, expect, it } from "vitest";
import { settingsValidationSchema } from "@/lib/validation/settings";

const mockT = ((key: string) => key) as TranslateFunction;

const definitions: SettingDefinition[] = [
  { key: "site_name", kind: "text", multiline: false },
  { key: "org_sameas_json", kind: "text", multiline: true },
  { key: "hero_cta_link", kind: "uri", multiline: false },
  { key: "org_logo", kind: "media", multiline: false, target: "org_logo" },
];

const schema = settingsValidationSchema(mockT, definitions);

describe("settingsValidationSchema", () => {
  // The schema is built from the server's registry, so it covers exactly the
  // keys the server sent rather than a second hard-coded list.
  it("accepts every field left blank", async () => {
    await expect(
      schema.isValid({
        site_name: "",
        org_sameas_json: "",
        hero_cta_link: "",
        org_logo: "",
      }),
    ).resolves.toBe(true);
  });

  it("accepts filled-in values", async () => {
    await expect(
      schema.isValid({
        site_name: "PixaBros",
        org_sameas_json: '["https://a.dev"]',
        hero_cta_link: "https://example.com/play",
        org_logo: "0123456789abcdef01234567",
      }),
    ).resolves.toBe(true);
  });

  it("requires a uri setting to be a full URL", async () => {
    for (const value of ["example.com", "/relative", "not a url"]) {
      await expect(schema.isValid({ hero_cta_link: value })).resolves.toBe(
        false,
      );
    }
  });

  it("does not constrain text settings", async () => {
    await expect(
      schema.isValid({ site_name: "anything at all !@#" }),
    ).resolves.toBe(true);
  });

  it("builds a schema for whatever definitions it is given", async () => {
    const other = settingsValidationSchema(mockT, [
      { key: "future_key", kind: "uri", multiline: false },
    ]);

    await expect(other.isValid({ future_key: "nope" })).resolves.toBe(false);
    await expect(other.isValid({ future_key: "https://a.dev" })).resolves.toBe(
      true,
    );
  });
});

describe("uri_list settings", () => {
  const withList = settingsValidationSchema(mockT, [
    { key: "org_sameas_json", kind: "uri_list", multiline: false },
  ]);

  it("accepts a list of full URLs", async () => {
    await expect(
      withList.isValid({
        org_sameas_json: ["https://twitter.com/x", "https://github.com/x"],
      }),
    ).resolves.toBe(true);
  });

  it("accepts an empty list", async () => {
    await expect(withList.isValid({ org_sameas_json: [] })).resolves.toBe(true);
  });

  // Each entry is validated on its own, so the wrong row is the row that
  // shows an error.
  it("rejects an entry that is not a full URL", async () => {
    await expect(
      withList.isValid({ org_sameas_json: ["https://ok.dev", "example.com"] }),
    ).resolves.toBe(false);
  });
});

describe("addresses", () => {
  const schema = settingsValidationSchema(mockT, [
    { key: "site_url", kind: "uri", multiline: false },
    { key: "hero_cta_link", kind: "link", multiline: false },
  ]);

  // The site runs on localhost while it is being built, and refusing to save a
  // real address is a worse failure than letting an odd one through.
  it.each([
    "http://localhost:8080",
    "https://pixabros.com",
    "https://pixabros.com/",
    "http://127.0.0.1:3000",
  ])("accepts %s as a site address", async (site_url) => {
    await expect(schema.isValid({ site_url })).resolves.toBe(true);
  });

  it.each([
    "pixabros.com",
    "/games",
    "not a url",
    "ftp://pixabros.com",
  ])("rejects %s as a site address", async (site_url) => {
    await expect(schema.isValid({ site_url })).resolves.toBe(false);
  });

  // A link is usually a page of this site, so a path is what gets typed.
  it.each([
    "/games",
    "/games/dungrid-tactics",
    "https://example.com/play",
    "",
  ])("accepts %s as a link", async (hero_cta_link) => {
    await expect(schema.isValid({ hero_cta_link })).resolves.toBe(true);
  });

  it.each([
    "games",
    "not a url",
    "//evil.test/games",
  ])("rejects %s as a link", async (hero_cta_link) => {
    await expect(schema.isValid({ hero_cta_link })).resolves.toBe(false);
  });
});
