interface SidebarItemData {
  id: string;
  icon?: IconElement;
  labelKey: string;
  path?: string;
  badge?: string | number;
  tooltip?: string;
  allowedRoles?: string[];
  children?: SidebarItemData[];
}

interface SidebarGroupData {
  titleKey?: string;
  items: SidebarItemData[];
}
