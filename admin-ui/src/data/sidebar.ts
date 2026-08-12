import {
  IconArticle,
  IconAward,
  IconDashboard,
  IconDeviceGamepad2,
  IconHome,
  IconMail,
  IconPhoto,
  IconRefresh,
  IconSettings,
  IconUsers,
} from "@tabler/icons-react";

// Items without a `path` are modules whose backend does not exist yet; the
// Sidebar renders those as inert, greyed-out entries. Give an item its path
// as soon as its screens land.
export const sidebarGroups: SidebarGroupData[] = [
  {
    items: [
      {
        id: "dashboard",
        icon: IconDashboard,
        labelKey: "menu.dashboard",
        path: "/",
      },
    ],
  },
  {
    titleKey: "menu.groups.content",
    items: [
      {
        id: "games",
        icon: IconDeviceGamepad2,
        labelKey: "menu.games",
        path: "/games",
      },
      {
        id: "devlog",
        icon: IconArticle,
        labelKey: "menu.devlog",
      },
      {
        id: "awards",
        icon: IconAward,
        labelKey: "menu.awards",
      },
      {
        id: "members",
        icon: IconUsers,
        labelKey: "menu.members",
        path: "/members",
      },
    ],
  },
  {
    titleKey: "menu.groups.site",
    items: [
      {
        id: "homepage",
        icon: IconHome,
        labelKey: "menu.homepage",
      },
      {
        id: "site-settings",
        icon: IconSettings,
        labelKey: "menu.siteSettings",
      },
      {
        id: "media",
        icon: IconPhoto,
        labelKey: "menu.media",
      },
    ],
  },
  {
    titleKey: "menu.groups.system",
    items: [
      {
        id: "contact",
        icon: IconMail,
        labelKey: "menu.contact",
      },
      {
        id: "regen-jobs",
        icon: IconRefresh,
        labelKey: "menu.regenJobs",
      },
    ],
  },
];
