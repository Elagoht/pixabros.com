// Only the public site's scripts. admin-ui is its own npm project with its own
// vitest, dependencies and tsconfig; pointing one runner at both would mean
// one config serving two unrelated builds.
export default {
  test: {
    include: ["internal/site/assets/**/*.test.js"],
  },
};
