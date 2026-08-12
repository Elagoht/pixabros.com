import { IconPhoneCall } from "@tabler/icons-react";
import type { FC } from "react";
import { Alert, Badge, Modal } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { formatDate } from "@/utilities/localization";

interface SubmissionDetailProps {
  submission: ResponseContactSubmission | null;
  onClose: () => void;
}

const SubmissionDetail: FC<SubmissionDetailProps> = ({
  submission,
  onClose,
}) => {
  const { t } = useI18n();

  if (!submission) {
    return null;
  }

  const rows: { label: string; value: string }[] = [
    { label: t("contact.detail.name"), value: submission.name },
    { label: t("contact.detail.email"), value: submission.email },
    { label: t("contact.detail.phone"), value: submission.phone },
    {
      label: t("contact.detail.received"),
      value: formatDate(submission.created_at, { format: "datetime" }),
    },
    // Kept for spam triage: it is the only thing here the sender did not type.
    { label: t("contact.detail.ipAddress"), value: submission.ip_address },
  ].filter((row) => row.value);

  return (
    <Modal open onClose={onClose} className="w-full max-w-2xl">
      <Modal.Header onClose={onClose}>
        <h2 className="min-w-0 truncate text-base font-semibold text-gray-800 dark:text-gray-100">
          {submission.subject}
        </h2>
      </Modal.Header>

      <Modal.Body className="space-y-4">
        {submission.wants_callback && (
          <Alert
            variant="warning"
            icon={IconPhoneCall}
            description={t("contact.detail.wantsCallback")}
          />
        )}

        {rows.length > 0 ? (
          <dl className="grid gap-x-4 gap-y-2 sm:grid-cols-[auto_1fr]">
            {rows.map((row) => (
              <div
                key={row.label}
                className="sm:col-span-2 sm:grid sm:grid-cols-subgrid"
              >
                <dt className="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {row.label}
                </dt>
                <dd className="break-all text-sm text-gray-800 dark:text-gray-100">
                  {row.value}
                </dd>
              </div>
            ))}
          </dl>
        ) : (
          <Badge variant="secondary">{t("contact.detail.noContact")}</Badge>
        )}

        <div className="space-y-1.5">
          <p className="text-xs font-medium text-gray-500 dark:text-gray-400">
            {t("contact.detail.message")}
          </p>
          {/* The sender's own words: shown as plain text, preserving their
              line breaks, never rendered as markup. */}
          <p className="whitespace-pre-wrap rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-800 dark:border-gray-700 dark:bg-gray-800/40 dark:text-gray-100">
            {submission.message}
          </p>
        </div>
      </Modal.Body>
    </Modal>
  );
};

export default SubmissionDetail;
