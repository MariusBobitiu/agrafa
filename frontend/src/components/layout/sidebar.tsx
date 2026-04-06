import { motion } from "framer-motion";
import { Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils.ts";
import { AnimateIcon } from "@/components/animate-ui/icons/icon.tsx";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { Separator } from "@/components/ui/separator.tsx";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip.tsx";
import { ProjectSwitcher } from "@/components/project/project-switcher.tsx";
import { useUIStore } from "@/stores/ui-store.ts";
import {
	ActivityIcon,
	BellIcon,
	LayoutDashboardIcon,
	SlidersHorizontalIcon,
	UnplugIcon,
	PanelLeftCloseIcon,
	PanelLeftOpenIcon
} from "@/components/animate-ui/icons";
import { Button } from "../ui/button";

const NAV_ITEMS = [
	{ to: "/overview", label: "Overview", icon: LayoutDashboardIcon },
	{ to: "/nodes", label: "Nodes", icon: UnplugIcon },
	{ to: "/services", label: "Services", icon: ActivityIcon },
	{ to: "/alerts", label: "Alerts", icon: BellIcon },
	{ to: "/settings", label: "Settings", icon: SlidersHorizontalIcon },
];

export function Sidebar() {
	const sidebarOpen = useUIStore((s) => s.sidebarOpen);
	const location = useLocation();
	const toggleSidebar = useUIStore((s) => s.toggleSidebar);

	function isRouteActive(to: string) {
		return location.pathname === to || location.pathname.startsWith(`${to}/`);
	}

	return (
		<motion.aside
			animate={{ width: sidebarOpen ? 220 : 60 }}
			transition={{ duration: 0.2, ease: "easeInOut" }}
			className="flex h-screen flex-col border-r border-border bg-card overflow-hidden shrink-0"
		>
			{/* Project switcher */}
			<div className="px-2 py-2 flex items-center shrink-0">
				<ProjectSwitcher isSidebarOpen={sidebarOpen} />
			</div>

			{/* Navigation */}
			<ScrollArea className="flex-1 py-6 px-1">
				<nav className="space-y-1 px-1">
					{NAV_ITEMS.map(({ to, label, icon: Icon }) => (
						<Tooltip key={to} delayDuration={0}>
							<TooltipTrigger asChild>
								<AnimateIcon asChild animate={isRouteActive(to)} animateOnHover>
									<Link
										to={to}
										aria-current={isRouteActive(to) ? "page" : undefined}
										data-active={isRouteActive(to) ? "true" : "false"}
										className={cn(
											"flex items-center gap-3 rounded-md px-2 py-2 text-sm font-medium transition-colors",
											isRouteActive(to)
												? "bg-sidebar-primary text-sidebar-primary-foreground"
												: "text-muted-foreground hover:bg-sidebar-primary/50 hover:text-foreground",
											sidebarOpen
												? "justify-start px-4"
												: "justify-center px-2",
										)}
									>
										<Icon size={16} strokeWidth={2.5} className="shrink-0" />
										{sidebarOpen && (
											<motion.span
												animate={{ opacity: 1 }}
												initial={{ opacity: 0 }}
												className="truncate"
											>
												{label}
											</motion.span>
										)}
									</Link>
								</AnimateIcon>
							</TooltipTrigger>
							{!sidebarOpen && (
								<TooltipContent side="right">{label}</TooltipContent>
							)}
						</Tooltip>
					))}
				</nav>
			</ScrollArea>

			<Separator />
			{/*  */}
			<div className="px-2 py-2">
				<AnimateIcon asChild animateOnHover>
					<Button
						variant="ghost"
						size="icon"
						onClick={toggleSidebar}
						aria-label="Toggle sidebar"
						className="w-full flex justify-between px-3 hover:bg-foreground/10 hover:text-foreground!"
					>
						<span className={cn(!sidebarOpen && "sr-only")}>
							Toggle sidebar
						</span>
						{sidebarOpen ? (
							<PanelLeftCloseIcon size={16} />
						) : (
							<PanelLeftOpenIcon size={16} />
						)}
					</Button>
				</AnimateIcon>
			</div>
		</motion.aside>
	);
}
