import { Http } from "@/utilities/http";

export class DevlogService {
  // Sorting is done by the database; the column set is whitelisted there.
  static list = (sort?: DevlogSort): Promise<ResponseDevlogPost[]> => {
    const query = sort?.field
      ? `?sort=${encodeURIComponent(sort.field)}&dir=${sort.direction}`
      : "";
    return Http.get<ResponseDevlogPost[]>(`/api/admin/devlog${query}`, {
      silent: true,
    });
  };

  // Accepts an id or a slug, so a link built from a public URL resolves too.
  static get = (key: string): Promise<ResponseDevlogPost> =>
    Http.get<ResponseDevlogPost>(`/api/admin/devlog/${key}`, { silent: true });

  static create = (
    payload: RequestCreateDevlogPost,
  ): Promise<ResponseDevlogPost> =>
    Http.post<ResponseDevlogPost>("/api/admin/devlog", payload, {
      silent: true,
    });

  static update = (
    postId: string,
    payload: RequestUpdateDevlogPost,
  ): Promise<ResponseDevlogPost> =>
    Http.put<ResponseDevlogPost>(`/api/admin/devlog/${postId}`, payload, {
      silent: true,
    });

  static delete = (postId: string): Promise<void> =>
    Http.delete<void>(`/api/admin/devlog/${postId}`, { silent: true });
}
