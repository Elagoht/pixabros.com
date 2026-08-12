import { Http } from "@/utilities/http";

// Screenshot endpoints address a game by its id only; the archive upload
// addresses it by slug, because the extracted build lives at /play/{slug}/.
export class GameService {
  static list = (): Promise<ResponseGame[]> =>
    Http.get<ResponseGame[]>("/api/admin/games", { silent: true });

  static get = (gameId: string): Promise<ResponseGame> =>
    Http.get<ResponseGame>(`/api/admin/games/${gameId}`, { silent: true });

  static create = (payload: RequestCreateGame): Promise<ResponseGame> =>
    Http.post<ResponseGame>("/api/admin/games", payload, { silent: true });

  static update = (
    gameId: string,
    payload: RequestUpdateGame,
  ): Promise<ResponseGame> =>
    Http.put<ResponseGame>(`/api/admin/games/${gameId}`, payload, {
      silent: true,
    });

  static delete = (gameId: string): Promise<void> =>
    Http.delete<void>(`/api/admin/games/${gameId}`, { silent: true });

  static reorder = (ids: string[]): Promise<void> =>
    Http.put<void>("/api/admin/games/reorder", { ids }, { silent: true });

  static listScreenshots = (gameId: string): Promise<ResponseScreenshot[]> =>
    Http.get<ResponseScreenshot[]>(`/api/admin/games/${gameId}/screenshots`, {
      silent: true,
    });

  static addScreenshot = (
    gameId: string,
    payload: RequestAddScreenshot,
  ): Promise<ResponseScreenshot> =>
    Http.post<ResponseScreenshot>(
      `/api/admin/games/${gameId}/screenshots`,
      payload,
      { silent: true },
    );

  static removeScreenshot = (
    gameId: string,
    screenshotId: string,
  ): Promise<void> =>
    Http.delete<void>(
      `/api/admin/games/${gameId}/screenshots/${screenshotId}`,
      { silent: true },
    );

  static reorderScreenshots = (gameId: string, ids: string[]): Promise<void> =>
    Http.put<void>(
      `/api/admin/games/${gameId}/screenshots/reorder`,
      { ids },
      { silent: true },
    );

  static uploadBuild = (slug: string, file: File): Promise<void> => {
    const body = new FormData();
    body.append("file", file);
    return Http.post<void>(`/api/admin/games/${slug}/upload`, body, {
      silent: true,
    });
  };
}
