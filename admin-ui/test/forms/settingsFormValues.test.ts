import { describe, expect, it } from "vitest";
import { parseUrlList } from "@/forms/SettingsForm";

describe("parseUrlList", () => {
  it("parses the JSON the API stores", () => {
    expect(parseUrlList('["https://a.dev","https://b.dev"]')).toEqual([
      "https://a.dev",
      "https://b.dev",
    ]);
  });

  it("treats a blank value as no addresses", () => {
    expect(parseUrlList("")).toEqual([]);
    expect(parseUrlList("   ")).toEqual([]);
  });

  // The value used to be hand-written JSON, so the screen has to survive
  // whatever is already in the column rather than crashing on it.
  it("degrades to an empty list on malformed JSON", () => {
    expect(parseUrlList("[{")).toEqual([]);
    expect(parseUrlList("not json")).toEqual([]);
  });

  it("degrades to an empty list when the JSON is not an array", () => {
    expect(parseUrlList('{"a":1}')).toEqual([]);
  });

  it("drops entries that are not strings", () => {
    expect(parseUrlList('["https://a.dev",1,null,{"b":2}]')).toEqual([
      "https://a.dev",
    ]);
  });
});
