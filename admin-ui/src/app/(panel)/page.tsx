import type { FC } from "react";
import { Container } from "@/components/ui";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";

const MainPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.dashboard") }]);

  return <Container>Admin Dashboard</Container>;
};

export default MainPage;
