const stripTrailingSlash = (value: string): string => value.replace(/\/+$/, "");

export class Navigation {
  static readonly basePath = stripTrailingSlash(import.meta.env.BASE_URL);

  static toAppPath = (browserPath: string): string => {
    const { basePath } = Navigation;
    if (!(basePath && browserPath.startsWith(basePath))) {
      return browserPath;
    }
    return browserPath.slice(basePath.length) || "/";
  };

  static toBrowserPath = (appPath: string): string =>
    `${Navigation.basePath}${appPath}`;

  static currentAppPath = (): string =>
    Navigation.toAppPath(window.location.pathname) +
    window.location.search +
    window.location.hash;

  static redirectToLogin = (): void => {
    const current = Navigation.currentAppPath();
    if (current.startsWith("/login")) {
      return;
    }
    window.location.href = Navigation.toBrowserPath(
      `/login?next=${encodeURIComponent(current)}`,
    );
  };
}
