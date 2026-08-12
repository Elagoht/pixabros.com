import { IconChevronRight } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { Tooltip } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import { useUIStore } from "@/lib/stores/ui";

interface SidebarProps {
  className?: string;
  groups?: SidebarGroupData[];
  items?: SidebarItemData[];
}

const anyChildActive = (
  children: SidebarItemData[] | undefined,
  pathname: string,
): boolean =>
  children
    ? children.some(
        (c) =>
          (c.path &&
            (c.path === pathname || pathname.startsWith(`${c.path}/`))) ||
          anyChildActive(c.children, pathname),
      )
    : false;

interface SidebarItemRendererProps {
  item: SidebarItemData;
  level?: number;
  expandedId?: string | null;
  onExpandId?: (id: string | null) => void;
}

const SidebarItemRenderer: FC<SidebarItemRendererProps> = ({
  item,
  level = 0,
  expandedId,
  onExpandId,
}) => {
  const { t } = useI18n();
  const location = useLocation();
  const { setSidebarOpen } = useUIStore();

  const hasChildren = !!(item.children && item.children.length > 0);

  const isActive = item.path
    ? item.path === "/"
      ? location.pathname === "/"
      : location.pathname === item.path
    : false;

  const [localExpanded, setLocalExpanded] = useState(
    hasChildren &&
      (isActive || anyChildActive(item.children, location.pathname)),
  );

  const isControlled =
    level === 0 && expandedId !== undefined && onExpandId !== undefined;
  const expanded = isControlled ? expandedId === item.id : localExpanded;

  const toggleExpanded = () => {
    if (isControlled) {
      onExpandId(expanded ? null : item.id);
    } else {
      setLocalExpanded((v) => !v);
    }
  };

  const linkClasses = classNames(
    "group relative flex min-w-0 flex-1 items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-sm font-medium outline-none transition-all duration-200",
    hasChildren && "font-semibold",
    isActive
      ? "sidebar-item-active text-primary-600 dark:text-primary-400 shadow-md shadow-primary-500/20"
      : "sidebar-item-hover text-gray-600 dark:text-gray-400",
  );

  const content = (
    <>
      {item.icon && (
        <span className="relative z-10 shrink-0">
          <span
            className={classNames(
              "inline-flex rounded-md p-1",
              isActive
                ? "sidebar-icon-active"
                : "sidebar-icon-idle text-gray-500 dark:text-gray-400",
            )}
          >
            <item.icon size={16} />
          </span>
          {item.badge !== undefined && (
            <span className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[9px] font-medium text-white shadow-lg shadow-red-500/30">
              {typeof item.badge === "number" && item.badge > 9
                ? "9+"
                : item.badge}
            </span>
          )}
        </span>
      )}
      <span className="relative z-10 flex-1 truncate">
        {t(item.labelKey as TranslationKey)}
      </span>
      {hasChildren && (
        <IconChevronRight
          size={14}
          className={classNames(
            "relative z-10 shrink-0 text-gray-400 dark:text-gray-500 transition-transform duration-150",
            expanded && "rotate-90",
          )}
        />
      )}
      {item.badge !== undefined && !hasChildren && (
        <span className="relative z-10 ml-auto rounded-full bg-primary-600 px-2 py-0.5 text-xs font-medium text-primary-100 shadow-sm shadow-primary-500/10">
          {item.badge}
        </span>
      )}
    </>
  );

  const el = hasChildren ? (
    <button type="button" onClick={toggleExpanded} className={linkClasses}>
      {content}
    </button>
  ) : item.path ? (
    <Link
      to={item.path}
      onClick={() => setSidebarOpen(false)}
      className={linkClasses}
    >
      {content}
    </Link>
  ) : (
    <span className={linkClasses}>{content}</span>
  );

  return (
    <div>
      <div
        className="flex items-stretch transition-all duration-300"
        style={{ marginLeft: `${level}rem` }}
      >
        <Tooltip
          content={item.tooltip || t(item.labelKey as TranslationKey)}
          position="right"
          block
        >
          {el}
        </Tooltip>
      </div>
      <div
        className={classNames(
          "grid transition-all duration-200",
          expanded ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
        )}
      >
        <div className="overflow-y-hidden overflow-x-visible pt-1 pr-1">
          {hasChildren && (
            <div className="space-y-0.5">
              {item.children?.map((child) => (
                <SidebarItemRenderer
                  key={child.id}
                  item={child}
                  level={level + 1}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const SidebarGroupItems: FC<{ items: SidebarItemData[] }> = ({ items }) => {
  const location = useLocation();

  const [expandedId, setExpandedId] = useState<string | null>(() => {
    for (const item of items) {
      if (
        item.children?.length &&
        anyChildActive(item.children, location.pathname)
      ) {
        return item.id;
      }
    }
    return null;
  });

  return (
    <>
      {items.map((item) => (
        <SidebarItemRenderer
          key={item.id}
          item={item}
          level={0}
          expandedId={expandedId}
          onExpandId={setExpandedId}
        />
      ))}
    </>
  );
};

const Sidebar: FC<SidebarProps> = ({ className, groups, items }) => {
  const { t } = useI18n();

  return (
    <div
      className={classNames(
        "flex h-full bg-white dark:bg-gray-1000 min-w-64 shrink-0 flex-col gap-4 border-r border-gray-200 dark:border-gray-800/60 p-4 shadow-sm shadow-gray-200 dark:shadow-black/30 transition-all duration-300",
        className,
      )}
    >
      <nav className="flex flex-1 flex-col gap-2 overflow-y-auto pr-1">
        {groups ? (
          groups.map((group, i) => (
            <div key={group.titleKey ?? i} className="flex flex-col gap-0.5">
              {group.titleKey && (
                <span className="px-2.5 py-1 text-[10px] font-bold uppercase text-secondary-600 dark:text-secondary-400">
                  {t(group.titleKey as TranslationKey)}
                </span>
              )}
              <SidebarGroupItems items={group.items} />
            </div>
          ))
        ) : items ? (
          <SidebarGroupItems items={items} />
        ) : null}
      </nav>
    </div>
  );
};

export default Sidebar;
