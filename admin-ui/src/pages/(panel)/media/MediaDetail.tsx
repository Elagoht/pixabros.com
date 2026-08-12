import { IconTrash } from "@tabler/icons-react";
import { type FC, useState } from "react";
import { Alert, Badge, Button, ImagePreview, Modal } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { formatDate } from "@/utilities/localization";

interface MediaDetailProps {
  item: ResponseMediaItem | null;
  isSaving: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onSaveAltText: (altText: string) => void;
  onDelete: () => void;
}

const MediaDetail: FC<MediaDetailProps> = ({
  item,
  isSaving,
  isDeleting,
  onClose,
  onSaveAltText,
  onDelete,
}) => {
  const { t } = useI18n();
  const [altText, setAltText] = useState(item?.alt_text ?? "");

  if (!item) {
    return null;
  }

  const inUse = item.usages.length > 0;

  return (
    <Modal open onClose={onClose} className="w-full max-w-2xl">
      <Modal.Header onClose={onClose}>
        <h2 className="text-base font-semibold text-gray-800 dark:text-gray-100">
          {t("media.detail.title")}
        </h2>
      </Modal.Header>

      <Modal.Body className="space-y-4">
        <ImagePreview
          src={item.url}
          alt={item.alt_text || t("media.altText.missing")}
          className="max-h-64 object-contain"
        />

        <dl className="grid gap-x-4 gap-y-2 text-sm sm:grid-cols-[auto_1fr]">
          <dt className="text-xs font-medium text-gray-500 dark:text-gray-400">
            {t("media.detail.dimensions")}
          </dt>
          <dd className="tabular-nums text-gray-800 dark:text-gray-100">
            {item.width} × {item.height}
          </dd>
          <dt className="text-xs font-medium text-gray-500 dark:text-gray-400">
            {t("media.detail.format")}
          </dt>
          <dd className="text-gray-800 dark:text-gray-100">{item.format}</dd>
          <dt className="text-xs font-medium text-gray-500 dark:text-gray-400">
            {t("media.detail.uploaded")}
          </dt>
          <dd className="text-gray-800 dark:text-gray-100">
            {formatDate(item.created_at, { format: "datetime" })}
          </dd>
        </dl>

        <div className="space-y-1.5">
          <p className="text-xs font-medium text-gray-500 dark:text-gray-400">
            {inUse ? t("media.usedIn") : t("media.unused")}
          </p>
          {inUse ? (
            <span className="flex flex-wrap gap-1">
              {item.usages.map((usage, index) => (
                <Badge
                  key={`${usage.module}-${usage.label}-${index}`}
                  variant="outline"
                >
                  {t(`media.modules.${usage.module}` as TranslationKey)} ·{" "}
                  {usage.label}
                </Badge>
              ))}
            </span>
          ) : (
            <Alert variant="info" description={t("media.sweepNote")} />
          )}
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="media-alt-text"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {t("media.altText.label")}
          </label>
          <textarea
            id="media-alt-text"
            rows={2}
            value={altText}
            onChange={(e) => setAltText(e.target.value)}
            className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-900 outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700 dark:bg-gray-800/50 dark:text-gray-50"
          />
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {t("media.altText.help")}
          </p>
        </div>
      </Modal.Body>

      <Modal.Footer className="justify-between gap-2">
        {/* Deleting an image that is still used would blank a page, so the
            control is disabled rather than failing on submit. */}
        <Button
          variant="destructive"
          size="sm"
          leftIcon={IconTrash}
          disabled={inUse || isDeleting}
          title={inUse ? t("media.delete.blocked") : t("media.delete.title")}
          onClick={onDelete}
        >
          {t("common.delete")}
        </Button>
        <Button
          variant="default"
          size="sm"
          disabled={isSaving || altText === item.alt_text}
          onClick={() => onSaveAltText(altText)}
        >
          {isSaving ? t("common.loading") : t("media.altText.save")}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default MediaDetail;
