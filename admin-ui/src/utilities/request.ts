import { toast } from "sonner";
import { useI18n } from "@/lib/stores/i18n";
import { ApiError } from "@/utilities/api-error";

type ErrorMessageMap = Partial<Record<number, TranslationKey>>;

interface ApiCallConfig<T = unknown> {
  onSuccess?: (data: T) => void;
  onError?: (error: ApiError) => void;
  onFinally?: () => void;
  successMessage?: TranslationKey;
  showSuccessMessage?: boolean;
  errorMessages?: ErrorMessageMap;
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
}

interface HandleRequestResult<T> {
  data?: T;
  error?: ApiError;
  success: boolean;
}

const getErrorKey = (
  error: unknown,
  errorMessages?: ErrorMessageMap,
): TranslationKey => {
  if (error instanceof ApiError && errorMessages?.[error.status]) {
    return errorMessages[error.status] as TranslationKey;
  }
  return "common.error";
};

export const handleRequest = async <T>(
  apiCall: () => Promise<T>,
  config: ApiCallConfig<T> = {},
): Promise<HandleRequestResult<T>> => {
  const {
    onSuccess,
    onError,
    onFinally,
    successMessage,
    showSuccessMessage,
    errorMessages,
    method = "GET",
  } = config;
  const { t } = useI18n.getState();

  const shouldShowSuccess =
    showSuccessMessage === undefined ? method !== "GET" : showSuccessMessage;

  try {
    const data = await apiCall();

    if (onSuccess) {
      onSuccess(data);
    }

    if (shouldShowSuccess && successMessage) {
      toast.success(t(successMessage));
    }

    return { data, success: true };
  } catch (err) {
    if (err instanceof ApiError) {
      const key = getErrorKey(err, errorMessages);
      toast.error(t(key));

      if (onError) {
        onError(err);
      }

      return { error: err, success: false };
    }

    throw err;
  } finally {
    if (onFinally) {
      onFinally();
    }
  }
};
