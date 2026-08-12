/** biome-ignore-all lint/a11y/noStaticElementInteractions: menu close event is actually required */
/** biome-ignore-all lint/a11y/useKeyWithClickEvents: menu close event is actually required */
import type { FC } from "react";
import { Outlet } from "react-router-dom";
import AuthGuard from "@/components/guards/AuthGuard";
import Header from "@/components/layout/Header";
import { Breadcrumb, Sidebar } from "@/components/ui";
import { sidebarGroups } from "@/data/sidebar";
import { useBreadcrumbStore } from "@/lib/stores/breadcrumb";
import { useI18n } from "@/lib/stores/i18n";
import { useUIStore } from "@/lib/stores/ui";

const PanelLayout: FC = () => {
	const { t } = useI18n();
	const { sidebarOpen, setSidebarOpen } = useUIStore();
	const breadcrumbItems = useBreadcrumbStore((s) => s.items);

	const closeSidebar = () => setSidebarOpen(false);

	return (
		<AuthGuard>
			<div className="flex h-screen flex-col overflow-hidden">
				<Header logo="/assets/logo.svg" appName={t("app.name")} />

				<div className="flex flex-1 overflow-hidden">
					<div
						className={`fixed top-14 inset-x-0 bottom-0 z-50 bg-black/50 transition-opacity lg:hidden ${sidebarOpen ? "opacity-100" : "opacity-0 pointer-events-none"}`}
						onClick={closeSidebar}
					/>

					<Sidebar
						className={`fixed top-14 bottom-0 left-0 z-50 transform transition-transform duration-300 lg:relative lg:top-0 lg:translate-x-0 lg:min-w-64 lg:shrink-0 lg:shadow-none ${sidebarOpen ? "translate-x-0 w-64" : "-translate-x-full lg:w-64"}`}
						groups={sidebarGroups}
					/>

					<main className="flex flex-1 flex-col overflow-hidden">
						<div className="min-h-0 flex-1 overflow-y-auto">
							{breadcrumbItems.length > 0 && (
								<div className="px-6 pt-4 pb-0">
									<Breadcrumb>
										{breadcrumbItems.map((item) => (
											<Breadcrumb.Item key={item.to} to={item.to}>
												{item.label}
											</Breadcrumb.Item>
										))}
									</Breadcrumb>
								</div>
							)}
							<Outlet />
						</div>
					</main>
				</div>
			</div>
		</AuthGuard>
	);
};

export default PanelLayout;
