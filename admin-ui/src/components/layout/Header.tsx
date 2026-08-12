import { IconKey, IconLogout, IconMenu2, IconUser } from "@tabler/icons-react";
import type { FC } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Button, Dropdown, Image } from "@/components/ui";
import { useAuthStore } from "@/lib/stores/auth";
import { useI18n } from "@/lib/stores/i18n";
import { useUIStore } from "@/lib/stores/ui";

interface HeaderProps {
	logo: string;
	appName?: string;
}

const Header: FC<HeaderProps> = ({ logo, appName }) => {
	const navigate = useNavigate();
	const { t } = useI18n();
	const { user, logout } = useAuthStore();
	const { toggleSidebar } = useUIStore();

	const userMenuItems = [
		{
			id: "username",
			label: user?.username ?? "",
			icon: IconUser,
			disabled: true,
		},
		{ id: "sep-user", label: undefined, separator: true },
		{
			id: "change-password",
			label: t("pages.changePassword.title"),
			icon: IconKey,
			onClick: () => navigate("/change-password"),
		},
		{ id: "sep", label: undefined, separator: true },
		{
			id: "logout",
			label: t("auth.logout"),
			icon: IconLogout,
			danger: true,
			onClick: () => logout(),
		},
	];

	return (
		<div className="relative flex h-14 w-full items-stretch bg-white dark:bg-gray-1000 shadow-sm shadow-gray-200/80 dark:shadow-black/30 lg:h-16">
			<div className="absolute inset-x-0 bottom-0 h-px bg-gray-200 dark:bg-gray-700/60" />

			<Button
				variant="ghost"
				onClick={toggleSidebar}
				className="shrink-0 !p-4 !rounded-none text-gray-600 dark:text-gray-400 hover:!bg-gray-100 dark:hover:!bg-white/5 hover:text-primary-600 dark:hover:text-primary-400 lg:hidden"
			>
				<IconMenu2 size={20} />
			</Button>

			{(logo || appName) && (
				<Link
					to="/"
					className="hidden min-w-64 shrink-0 items-center justify-center gap-3 border-r border-gray-200 dark:border-gray-800/60 px-4 transition-colors hover:bg-gray-50 dark:hover:bg-gray-900/50 lg:flex"
				>
					<Image
						src={logo}
						alt={t("common.logo")}
						width={44}
						height={44}
						className="rounded-lg shadow-md shadow-gray-300/30 dark:shadow-gray-500/15"
					/>
					{appName && (
						<span className="text-lg font-bold text-gray-800 dark:text-gray-100">
							{appName}
						</span>
					)}
				</Link>
			)}

			<div className="flex flex-1 items-center justify-center px-4"></div>

			<div className="flex items-center justify-end px-4">
				<Dropdown
					align="right"
					trigger={
						<Button
							variant="ghost"
							className="!rounded-xl !p-1.5 hover:!bg-gray-100 dark:hover:!bg-white/10 hover:shadow-md hover:shadow-primary-500/15"
						>
							<div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-500 text-white shadow-md shadow-primary-500/30">
								<IconUser size={16} />
							</div>
						</Button>
					}
					items={userMenuItems}
				/>
			</div>
		</div>
	);
};

export default Header;
