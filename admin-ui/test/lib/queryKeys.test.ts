import { describe, expect, it } from "vitest";
import { queryKeys } from "@/lib/query/keys";

describe("queryKeys", () => {
  it("scopes every game key under a shared prefix so one invalidation clears them", () => {
    const prefix = queryKeys.games.all;

    for (const key of [
      queryKeys.games.list(),
      queryKeys.games.detail("g1"),
      queryKeys.games.screenshots("g1"),
    ]) {
      expect(key.slice(0, prefix.length)).toEqual([...prefix]);
    }
  });

  it("keys different games separately", () => {
    expect(queryKeys.games.detail("a")).not.toEqual(queryKeys.games.detail("b"));
    expect(queryKeys.games.screenshots("a")).not.toEqual(
      queryKeys.games.screenshots("b"),
    );
  });

  it("does not collide a game's detail with its screenshots", () => {
    expect(queryKeys.games.detail("a")).not.toEqual(
      queryKeys.games.screenshots("a"),
    );
  });

  it("keeps media keys out of the games namespace", () => {
    expect(queryKeys.media.detail("m1")[0]).not.toBe(queryKeys.games.all[0]);
  });
});
