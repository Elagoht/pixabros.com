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
    expect(queryKeys.games.detail("a")).not.toEqual(
      queryKeys.games.detail("b"),
    );
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

describe("queryKeys.games list sorting", () => {
  it("keys each ordering separately so a re-sort refetches", () => {
    expect(
      queryKeys.games.list({ field: "title", direction: "asc" }),
    ).not.toEqual(queryKeys.games.list({ field: "title", direction: "desc" }));
    expect(
      queryKeys.games.list({ field: "title", direction: "asc" }),
    ).not.toEqual(queryKeys.games.list({ field: "slug", direction: "asc" }));
  });

  // Invalidating after a mutation has to clear every cached ordering, not
  // just the one currently on screen.
  it("shares one prefix across every ordering", () => {
    const prefix = queryKeys.games.lists();
    for (const sort of [
      undefined,
      { field: "title", direction: "asc" } as const,
      { field: "display_order", direction: "desc" } as const,
    ]) {
      expect(queryKeys.games.list(sort).slice(0, prefix.length)).toEqual([
        ...prefix,
      ]);
    }
  });
});
