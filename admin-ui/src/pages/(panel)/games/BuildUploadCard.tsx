import { IconExternalLink, IconUpload } from "@tabler/icons-react";
import { type ChangeEvent, type FC, useRef, useState } from "react";
import { Alert, Button, Card } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { GameService } from "@/services/game";
import { handleRequest } from "@/utilities/request";

interface BuildUploadCardProps {
  slug: string;
  webExportPath: string;
  onUploaded: () => void;
}

const BuildUploadCard: FC<BuildUploadCardProps> = ({
  slug,
  webExportPath,
  onUploaded,
}) => {
  const { t } = useI18n();
  const inputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);

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
      onUploaded();
    }
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  return (
    <Card>
      <Card.Header>
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

        <Button
          variant="outline"
          size="sm"
          leftIcon={IconUpload}
          disabled={isUploading}
          className="w-full"
          onClick={() => inputRef.current?.click()}
        >
          {isUploading ? t("games.build.uploading") : t("games.build.upload")}
        </Button>

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
    </Card>
  );
};

export default BuildUploadCard;
