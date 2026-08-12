import {
  IconArticle,
  IconAward,
  IconDeviceGamepad2,
  IconMail,
  IconPhoto,
  IconUsers,
} from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";
import { Alert, Container } from "@/components/ui";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import { queryKeys } from "@/lib/query/keys";
import { useI18n } from "@/lib/stores/i18n";
import { StatsService } from "@/services/stats";
import QuickActions from "./QuickActions";
import StatCard from "./StatCard";

const DashboardView: FC = () => {
  const { t } = useI18n();

  useBreadcrumb([{ label: t("menu.dashboard") }]);

  const {
    data: stats,
    isLoading,
    isError,
  } = useQuery({
    queryKey: queryKeys.stats.all,
    queryFn: StatsService.get,
  });

  return (
    <Container className="space-y-6 py-6">
      {isError && (
        <Alert
          variant="error"
          title={t("pages.dashboard.loadError")}
          description={t("pages.dashboard.loadErrorDescription")}
        />
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {/* Unread messages come first and are the only highlighted card: it is
            the one number on this page that is a task rather than a fact. */}
        <StatCard
          icon={IconMail}
          label={t("pages.dashboard.unreadMessages")}
          value={stats?.contact.unread}
          hint={t("pages.dashboard.totalMessages", {
            count: String(stats?.contact.total ?? 0),
          })}
          to="/contact"
          accent="rose"
          loading={isLoading}
          highlight={(stats?.contact.unread ?? 0) > 0}
        />

        <StatCard
          icon={IconDeviceGamepad2}
          label={t("pages.dashboard.games")}
          value={stats?.games.total}
          hint={t("pages.dashboard.gamesHint", {
            published: String(stats?.games.published ?? 0),
            playable: String(stats?.games.playable ?? 0),
          })}
          to="/games"
          accent="primary"
          loading={isLoading}
        />

        <StatCard
          icon={IconArticle}
          label={t("pages.dashboard.devlog")}
          value={stats?.devlog.total}
          hint={t("pages.dashboard.publishedCount", {
            count: String(stats?.devlog.published ?? 0),
          })}
          to="/devlog"
          accent="sky"
          loading={isLoading}
        />

        <StatCard
          icon={IconAward}
          label={t("pages.dashboard.awards")}
          value={stats?.awards}
          to="/awards"
          accent="amber"
          loading={isLoading}
        />

        <StatCard
          icon={IconUsers}
          label={t("pages.dashboard.members")}
          value={stats?.members}
          to="/members"
          accent="emerald"
          loading={isLoading}
        />

        <StatCard
          icon={IconPhoto}
          label={t("pages.dashboard.media")}
          value={stats?.media}
          to="/media"
          accent="violet"
          loading={isLoading}
        />
      </div>

      <QuickActions />
    </Container>
  );
};

export default DashboardView;
