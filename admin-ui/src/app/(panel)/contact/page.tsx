import type { FC } from "react";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import ContactListView from "@/pages/(panel)/contact/ContactListView";

const ContactPage: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.contact") }]);

  return <ContactListView />;
};

export default ContactPage;
