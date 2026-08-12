import {
  IconDeviceGamepad2,
  IconExternalLink,
  IconTrash,
  IconUpload,
} from "@tabler/icons-react";
import { type ChangeEvent, type FC, useRef, useState } from "react";
import { Alert, Button, Card, Dialog } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";

interface BuildUploadCardProps {
  gameId: string;
  slug: string;
  webExportPath: string;
  onChanged: () => void;
}

const BuildUploadCard: FC<BuildUploadCardProps> = ({
  gameId,
  slug,
  webExportPath,
  onChanged,
}) => {
  const { t } = useI18n();
  const inputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [confirmingRemove, setConfirmingRemove] = useState(false);

  const handleFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    setIsUploading(true);
    const { success } = await handleRequest(
      () => GameService.uploadBuild(slug, file),
      {
        method: "POST",
        errorMessages: {
          400: "games.errors.invalidArchive",
          413: "games.errors.fileTooLarge",
        },
        successMessage: "games.toast.buildUploaded",
        onFinally: () => setIsUploading(false),
      },
    );
    if (success) {
      onChanged();
    }
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  const removeBuild = async () => {
    setIsRemoving(true);
    const { success } = await handleRequest(
      () => GameService.deleteBuild(gameId),
      {
        method: "DELETE",
        successMessage: "games.toast.buildRemoved",
        onFinally: () => setIsRemoving(false),
      },
    );
    setConfirmingRemove(false);
    if (success) {
      onChanged();
    }
  };

  return (
    <Card>
      <Card.Header icon={IconDeviceGamepad2}>
        <h2 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
          {t("games.build.title")}
        </h2>
      </Card.Header>

      <Card.Body className="space-y-3">
        {webExportPath ? (
          <div className="space-y-2">
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {t("games.build.current")}
            </p>
            <p className="break-all font-mono text-xs text-gray-700 dark:text-gray-300">
              {webExportPath}
            </p>
            {/* /play/ is served by Go outside the SPA's basename, so this has
                to be a real anchor, not a react-router Link. */}
            <a
              href={`/play/${slug}/`}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
            >
              <IconExternalLink size={14} />
              {t("games.build.play")}
            </a>
          </div>
        ) : (
          <Alert variant="info" description={t("games.build.empty")} />
        )}

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            leftIcon={IconUpload}
            disabled={isUploading || isRemoving}
            className="flex-1"
            onClick={() => inputRef.current?.click()}
          >
            {isUploading
              ? t("games.build.uploading")
              : webExportPath
                ? t("games.build.replace")
                : t("games.build.upload")}
          </Button>

          {webExportPath && (
            <Button
              variant="ghost"
              size="sm"
              title={t("games.build.remove")}
              disabled={isUploading || isRemoving}
              onClick={() => setConfirmingRemove(true)}
            >
              <IconTrash size={14} />
            </Button>
          )}
        </div>

        <p className="text-[10px] text-gray-400 dark:text-gray-500">
          {t("games.build.help")}
        </p>

        <input
          ref={inputRef}
          type="file"
          accept=".zip,.tar,.tar.gz,.tgz"
          className="hidden"
          onChange={handleFile}
        />
      </Card.Body>

      <Dialog
        open={confirmingRemove}
        onClose={() => setConfirmingRemove(false)}
        title={t("games.build.remove")}
        description={t("games.build.removeDescription")}
        confirmLabel={t("common.delete")}
        confirmVariant="destructive"
        onConfirm={removeBuild}
        onCancel={() => setConfirmingRemove(false)}
      />
    </Card>
  );
};

export default BuildUploadCard;
