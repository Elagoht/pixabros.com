import { Http } from "@/utilities/http";

export class AwardService {
  // Sorting is done by the database; the column set is whitelisted there.
  static list = (sort?: AwardSort): Promise<ResponseAward[]> => {
    const query = sort?.field
      ? `?sort=${encodeURIComponent(sort.field)}&dir=${sort.direction}`
      : "";
    return Http.get<ResponseAward[]>(`/api/admin/awards${query}`, {
      silent: true,
    });
  };

  static get = (awardId: string): Promise<ResponseAward> =>
    Http.get<ResponseAward>(`/api/admin/awards/${awardId}`, { silent: true });

  static create = (payload: RequestCreateAward): Promise<ResponseAward> =>
    Http.post<ResponseAward>("/api/admin/awards", payload, { silent: true });

  static update = (
    awardId: string,
    payload: RequestUpdateAward,
  ): Promise<ResponseAward> =>
    Http.put<ResponseAward>(`/api/admin/awards/${awardId}`, payload, {
      silent: true,
    });

  static delete = (awardId: string): Promise<void> =>
    Http.delete<void>(`/api/admin/awards/${awardId}`, { silent: true });
}
