import { Http } from "@/utilities/http";

export class SettingsService {
  static get = (group: SettingsGroupName): Promise<ResponseSettingsGroup> =>
    Http.get<ResponseSettingsGroup>(`/api/admin/settings/${group}`, {
      silent: true,
    });

  static update = (
    group: SettingsGroupName,
    payload: RequestUpdateSettings,
  ): Promise<ResponseSettingsGroup> =>
    Http.put<ResponseSettingsGroup>(`/api/admin/settings/${group}`, payload, {
      silent: true,
    });
}
