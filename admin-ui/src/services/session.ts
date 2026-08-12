import { Http } from "@/utilities/http";

// Every call is `silent` because callers route errors through
// `handleRequest`, which surfaces a translated message. Letting Http toast as
// well would stack a second, untranslated toast from the server.
export class SessionService {
  static create = (payload: RequestLogin): Promise<ResponseLogin> =>
    Http.post<ResponseLogin>("/api/admin/login", payload, {
      silent: true,
      skipAuthRedirect: true,
    });

  static delete = (): Promise<void> =>
    Http.post<void>("/api/admin/logout", undefined, {
      silent: true,
      skipAuthRedirect: true,
    });

  // A 401 here is the expected answer for a signed-out visitor, so it must
  // not bounce the browser to /login -- the guards decide what to render.
  static me = (): Promise<ResponseWhoami> =>
    Http.get<ResponseWhoami>("/api/admin/whoami", {
      silent: true,
      skipAuthRedirect: true,
    });

  static changePassword = (payload: RequestChangePassword): Promise<void> =>
    Http.post<void>("/api/admin/change-password", payload, { silent: true });
}
