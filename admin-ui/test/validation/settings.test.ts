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
