import { describe, expect, it } from "vitest";
import { memberValidationSchema } from "@/lib/validation/member";

const mockT = ((key: string) => key) as TranslateFunction;
const schema = memberValidationSchema(mockT);

const valid: MemberFormValues = {
  name: "Furkan",
  tags: "",
  description: "",
  links: [],
  is_published: false,
};

describe("memberValidationSchema", () => {
  it("accepts a minimal member", async () => {
    await expect(schema.isValid(valid)).resolves.toBe(true);
  });

  it("requires a name", async () => {
    await expect(schema.isValid({ ...valid, name: "" })).resolves.toBe(false);
  });

  it("rejects a whitespace-only name, matching the server's trim", async () => {
    await expect(schema.isValid({ ...valid, name: "   " })).resolves.toBe(
      false,
    );
  });

  it("accepts a fully filled link", async () => {
    await expect(
      schema.isValid({
        ...valid,
        links: [{ label: "GitHub", url: "https://github.com/x" }],
      }),
    ).resolves.toBe(true);
  });

  // Half a link renders as a broken link on the public site.
  it("rejects a half-filled link", async () => {
    await expect(
      schema.isValid({ ...valid, links: [{ label: "GitHub", url: "" }] }),
    ).resolves.toBe(false);
    await expect(
      schema.isValid({
        ...valid,
        links: [{ label: "", url: "https://a.dev" }],
      }),
    ).resolves.toBe(false);
  });
});
