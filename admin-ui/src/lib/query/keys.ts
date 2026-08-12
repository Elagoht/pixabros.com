// Query keys are built here rather than inline so an invalidation can never
// silently miss a cache entry because of a typo'd key array.
export const queryKeys = {
  games: {
    all: ["games"] as const,
    list: () => [...queryKeys.games.all, "list"] as const,
    detail: (gameId: string) =>
      [...queryKeys.games.all, "detail", gameId] as const,
    screenshots: (gameId: string) =>
      [...queryKeys.games.all, "screenshots", gameId] as const,
  },
  media: {
    all: ["media"] as const,
    detail: (mediaId: string) => [...queryKeys.media.all, mediaId] as const,
  },
};
