import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

let source = "";
try {
  source = readFileSync("internal/site/assets/share.js", "utf8");
} catch {}

const loadShare = () => {
  const window = { addEventListener() {} };
  const navigator = {};
  const document = { querySelectorAll: () => [] };
  new Function("window", "navigator", "document", source)(window, navigator, document);
  return window.ShareLogic;
};

describe("sharePost", () => {
  it("uses the device share sheet when available", async () => {
    const calls = [];
    const navigator = {
      share: async (payload) => calls.push(payload),
      clipboard: { writeText: async () => {} },
    };

    await loadShare().sharePost(navigator, {
      title: "A Sharp Update",
      url: "https://pixabros.com/devlog/sharp-update",
    });

    expect(calls).toEqual([
      {
        title: "A Sharp Update",
        url: "https://pixabros.com/devlog/sharp-update",
      },
    ]);
  });

  it("copies the canonical URL when native sharing is unavailable", async () => {
    const copied = [];
    const navigator = {
      clipboard: { writeText: async (value) => copied.push(value) },
    };

    const result = await loadShare().sharePost(navigator, {
      title: "A Sharp Update",
      url: "https://pixabros.com/devlog/sharp-update",
    });

    expect(copied).toEqual(["https://pixabros.com/devlog/sharp-update"]);
    expect(result).toBe("copied");
  });
});
