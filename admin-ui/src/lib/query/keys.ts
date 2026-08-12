// Query keys are built here rather than inline so an invalidation can never
// silently miss a cache entry because of a typo'd key array.
export const queryKeys = {
  games: {
    all: ["games"] as const,
    // lists() is the prefix every sorted list shares. Invalidating the exact
    // list(sort) key would only refresh the one ordering currently on screen
    // and leave every other cached ordering stale.
    lists: () => [...queryKeys.games.all, "list"] as const,
    list: (sort?: GameSort) =>
      [
        ...queryKeys.games.lists(),
        sort?.field ?? null,
        sort?.direction ?? null,
      ] as const,
    detail: (gameId: string) =>
      [...queryKeys.games.all, "detail", gameId] as const,
    screenshots: (gameId: string) =>
      [...queryKeys.games.all, "screenshots", gameId] as const,
  },
  members: {
    all: ["members"] as const,
    lists: () => [...queryKeys.members.all, "list"] as const,
    list: (sort?: MemberSort) =>
      [
        ...queryKeys.members.lists(),
        sort?.field ?? null,
        sort?.direction ?? null,
      ] as const,
    detail: (memberId: string) =>
      [...queryKeys.members.all, "detail", memberId] as const,
  },
  media: {
    all: ["media"] as const,
    detail: (mediaId: string) => [...queryKeys.media.all, mediaId] as const,
  },
};
