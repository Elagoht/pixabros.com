import { toast } from "sonner";
import { ApiError } from "@/utilities/api-error";
import { Environment } from "@/utilities/environment";
import { Navigation } from "@/utilities/navigation";

// The Go API answers every failure with {"error":{"code","message"}}.
const extractErrorMessage = (body: Record<string, unknown>): string | null => {
  const error = body.error;
  if (error && typeof error === "object") {
    const { message } = error as { message?: unknown };
    if (typeof message === "string" && message) {
      return message;
    }
  }
  if (typeof body.message === "string" && body.message) {
    return body.message;
  }
  return null;
};

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  skipAuthRedirect?: boolean;
  silent?: boolean;
}

type ReadOptions = Omit<RequestOptions, "body">;

const parseResponse = async <T>(response: Response): Promise<T> => {
  const contentType = response.headers.get("content-type");
  const contentLength = response.headers.get("content-length");

  if (!contentType?.includes("application/json") || contentLength === "0") {
    return undefined as T;
  }

  const text = await response.text();
  if (!text) {
    return undefined as T;
  }
  return JSON.parse(text) as T;
};

const request = async <T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> => {
  const { body, headers, skipAuthRedirect, silent, ...rest } = options;

  const isFormData = body instanceof FormData;

  const response = await fetch(`${Environment.apiBase}${path}`, {
    credentials: "include",
    headers: isFormData
      ? headers
      : { "Content-Type": "application/json", ...headers },
    body: isFormData ? body : body ? JSON.stringify(body) : undefined,
    ...rest,
  });

  if (!response.ok) {
    const errorBody = await response.json().catch(() => ({}));

    // The session cookie is HttpOnly, so an expired session is only ever
    // discovered here. There is no refresh endpoint -- sessions are absolute
    // and re-authenticating means logging in again.
    if (response.status === 401 && !skipAuthRedirect) {
      Navigation.redirectToLogin();
      throw new ApiError(401, errorBody);
    }

    if (!silent) {
      const message = extractErrorMessage(errorBody);
      if (message) {
        toast.error(message);
      }
    }
    throw new ApiError(response.status, errorBody);
  }

  return parseResponse<T>(response);
};

export const Http = {
  get: <T>(path: string, options?: ReadOptions) =>
    request<T>(path, { ...options, method: "GET" }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "POST", body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "PUT", body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "PATCH", body }),
  delete: <T>(path: string, options?: ReadOptions) =>
    request<T>(path, { ...options, method: "DELETE" }),
};
