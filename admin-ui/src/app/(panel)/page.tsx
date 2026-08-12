import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import type { FC } from "react";

const MainPage: FC = () => {
	const { t } = useI18n();

	useBreadcrumb([{ label: t("menu.dashboard") }]);

	return <>Admin Dashboard</>;
};

export default MainPage;
