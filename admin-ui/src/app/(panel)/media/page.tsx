import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import MediaListView from "@/pages/(panel)/media/MediaListView";

const MediaPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.media") }]);

  return <MediaListView />;
};

export default MediaPage;
