import { toast } from "sonner";
import { ApiError } from "@/utilities/api-error";
import { Environment } from "@/utilities/environment";

function extractErrorMessage(body: Record<string, unknown>): string | null {
  if (body.detail && typeof body.detail === "string") {
    return body.detail;
  }
  for (const [, value] of Object.entries(body)) {
    if (Array.isArray(value) && value.length > 0) {
      return String(value[0]);
    }
    if (typeof value === "string" && value) {
      return value;
    }
  }
  return null;
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  skipAuthRefresh?: boolean;
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

let refreshPromise: Promise<boolean> | null = null;

const tryRefresh = async (): Promise<boolean> => {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = fetch(`${Environment.apiBase}/user/token/refresh/`, {
    method: "POST",
    credentials: "include",
  })
    .then((res) => res.ok)
    .catch(() => false)
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
};

const request = async <T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> => {
  const { body, headers, skipAuthRefresh, ...rest } = options;

  const url = `${Environment.apiBase}${path}`;

  const isFormData = body instanceof FormData;

  const response = await fetch(url, {
    credentials: "include",
    headers: isFormData
      ? headers
      : { "Content-Type": "application/json", ...headers },
    body: isFormData ? body : body ? JSON.stringify(body) : undefined,
    ...rest,
  });

  if (response.status === 401 && !skipAuthRefresh) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      const retryResponse = await fetch(url, {
        credentials: "include",
        headers: isFormData
          ? headers
          : { "Content-Type": "application/json", ...headers },
        body: isFormData ? body : body ? JSON.stringify(body) : undefined,
        ...rest,
      });

      if (retryResponse.ok) {
        return parseResponse<T>(retryResponse);
      }

      const retryBody = await retryResponse.json().catch(() => ({}));
      if (!options.silent) {
        const retryMsg = extractErrorMessage(retryBody);
        if (retryMsg) {
          toast.error(retryMsg);
        }
      }
      throw new ApiError(retryResponse.status, retryBody);
    }

    if (window.location.pathname !== "/login") {
      const currentPath =
        window.location.pathname +
        window.location.search +
        window.location.hash;
      window.location.href = `/login?next=${encodeURIComponent(currentPath)}`;
    }
    throw new ApiError(401, {});
  }

  if (!response.ok) {
    const errorBody = await response.json().catch(() => ({}));
    if (!options.silent) {
      const msg = extractErrorMessage(errorBody);
      if (msg) {
        toast.error(msg);
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
