import { Http } from "@/utilities/http";

export class StatsService {
  static get = (): Promise<ResponseStats> =>
    Http.get<ResponseStats>("/api/admin/stats", { silent: true });
}
