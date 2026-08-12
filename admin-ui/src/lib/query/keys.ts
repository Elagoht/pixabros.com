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
  awards: {
    all: ["awards"] as const,
    lists: () => [...queryKeys.awards.all, "list"] as const,
    list: (sort?: AwardSort) =>
      [
        ...queryKeys.awards.lists(),
        sort?.field ?? null,
        sort?.direction ?? null,
      ] as const,
    detail: (awardId: string) =>
      [...queryKeys.awards.all, "detail", awardId] as const,
  },
  devlog: {
    all: ["devlog"] as const,
    lists: () => [...queryKeys.devlog.all, "list"] as const,
    list: (sort?: DevlogSort) =>
      [
        ...queryKeys.devlog.lists(),
        sort?.field ?? null,
        sort?.direction ?? null,
      ] as const,
    detail: (postId: string) =>
      [...queryKeys.devlog.all, "detail", postId] as const,
  },
  contact: {
    all: ["contact"] as const,
    lists: () => [...queryKeys.contact.all, "list"] as const,
    list: (sort?: ContactSort) =>
      [
        ...queryKeys.contact.lists(),
        sort?.field ?? null,
        sort?.direction ?? null,
      ] as const,
  },
  settings: {
    all: ["settings"] as const,
    group: (group: SettingsGroupName) =>
      [...queryKeys.settings.all, group] as const,
  },
  stats: {
    all: ["stats"] as const,
  },
  media: {
    all: ["media"] as const,
    library: () => [...queryKeys.media.all, "library"] as const,
    detail: (mediaId: string) => [...queryKeys.media.all, mediaId] as const,
  },
};
