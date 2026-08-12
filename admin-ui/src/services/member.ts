import { Http } from "@/utilities/http";

export class MemberService {
  // Sorting is done by the database; the column set is whitelisted there.
  static list = (sort?: MemberSort): Promise<ResponseMember[]> => {
    const query = sort?.field
      ? `?sort=${encodeURIComponent(sort.field)}&dir=${sort.direction}`
      : "";
    return Http.get<ResponseMember[]>(`/api/admin/members${query}`, {
      silent: true,
    });
  };

  static get = (memberId: string): Promise<ResponseMember> =>
    Http.get<ResponseMember>(`/api/admin/members/${memberId}`, {
      silent: true,
    });

  static create = (payload: RequestCreateMember): Promise<ResponseMember> =>
    Http.post<ResponseMember>("/api/admin/members", payload, { silent: true });

  static update = (
    memberId: string,
    payload: RequestUpdateMember,
  ): Promise<ResponseMember> =>
    Http.put<ResponseMember>(`/api/admin/members/${memberId}`, payload, {
      silent: true,
    });

  static delete = (memberId: string): Promise<void> =>
    Http.delete<void>(`/api/admin/members/${memberId}`, { silent: true });

  static reorder = (ids: string[]): Promise<void> =>
    Http.put<void>("/api/admin/members/reorder", { ids }, { silent: true });
}
