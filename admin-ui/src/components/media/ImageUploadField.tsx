import { IconPhotoPlus, IconTrash } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { type ChangeEvent, type FC, useRef, useState } from "react";
import { Button, Loading } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { MediaService } from "@/services/media";
import { handleRequest } from "@/utilities/request";

interface ImageUploadFieldProps {
  label: string;
  hint: string;
  target: MediaTarget;
  mediaId: string | null;
  onChange: (mediaId: string | null) => void;
}

const ImageUploadField: FC<ImageUploadFieldProps> = ({
  label,
  hint,
  target,
  mediaId,
  onChange,
}) => {
  const { t } = useI18n();
  const inputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);
  // The freshly uploaded URL wins over the query so the thumbnail swaps
  // immediately instead of waiting on a refetch.
  const [uploadedUrl, setUploadedUrl] = useState<string | null>(null);

  const { data: media } = useQuery({
    queryKey: queryKeys.media.detail(mediaId ?? ""),
    queryFn: () => MediaService.get(mediaId as string),
    enabled: !!mediaId,
  });

  const previewUrl = uploadedUrl ?? media?.url ?? null;

  const handleFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    setIsUploading(true);
    const { data } = await handleRequest(
      () => MediaService.upload(file, target),
      {
        method: "POST",
        errorMessages: {
          400: "media.errors.invalidImage",
          413: "media.errors.fileTooLarge",
        },
        showSuccessMessage: false,
        onFinally: () => setIsUploading(false),
      },
    );
    if (data) {
      setUploadedUrl(data.url);
      onChange(data.id);
    }
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  const clear = () => {
    setUploadedUrl(null);
    onChange(null);
  };

  return (
    <div className="space-y-2 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800/40">
      <p className="text-xs font-medium text-gray-700 dark:text-gray-300">
        {label}
      </p>

      <div className="flex h-28 items-center justify-center overflow-hidden rounded-md border border-dashed border-gray-300 bg-white dark:border-gray-600 dark:bg-gray-900">
        {isUploading ? (
          <Loading />
        ) : previewUrl ? (
          <img
            src={previewUrl}
            alt={label}
            className="max-h-full max-w-full object-contain"
          />
        ) : (
          <span className="text-xs text-gray-400 dark:text-gray-600">
            {t("media.empty")}
          </span>
        )}
      </div>

      <p className="text-[10px] text-gray-400 dark:text-gray-500">{hint}</p>

      <div className="flex gap-1.5">
        <Button
          variant="outline"
          size="sm"
          leftIcon={IconPhotoPlus}
          disabled={isUploading}
          className="flex-1"
          onClick={() => inputRef.current?.click()}
        >
          {previewUrl ? t("media.replace") : t("media.upload")}
        </Button>
        {previewUrl && (
          <Button
            variant="ghost"
            size="sm"
            disabled={isUploading}
            title={t("common.delete")}
            onClick={clear}
          >
            <IconTrash size={14} />
          </Button>
        )}
      </div>

      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={handleFile}
      />
    </div>
  );
};

export default ImageUploadField;
