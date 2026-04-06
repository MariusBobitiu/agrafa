import { useEffect } from "react";
import {
  Select,
  SelectContent,
  SelectGroup,
	SelectLabel,
	SelectSeparator,
} from "@/components/ui/select.tsx";
import { useProjects } from "@/hooks/use-projects.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { motion } from "motion/react";
import { SelectTrigger, SelectItem } from "@radix-ui/react-select";
import { cn } from "@/lib/utils";
import { ChevronUpDownIcon, PlusIcon, AnimateIcon } from "../animate-ui/icons";

export function ProjectSwitcher({
  isSidebarOpen = true,
} : {
  isSidebarOpen?: boolean;
}) {
  const { data } = useProjects();
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const setActiveProjectId = useUIStore((s) => s.setActiveProjectId);

  // Auto-select first project if none selected
  useEffect(() => {
    if (!activeProjectId && data?.projects.length) {
      setActiveProjectId(data.projects[0]!.id);
    }
  }, [data, activeProjectId, setActiveProjectId]);

  const projects = data?.projects ?? [];

  return (
		<Select
			value={activeProjectId?.toString() ?? ""}
			onValueChange={(val) => setActiveProjectId(Number(val))}
		>
			<SelectTrigger asChild>
				<AnimateIcon asChild animateOnHover>
					<motion.button
						animate={{ opacity: projects.length ? 1 : 0.5 }}
						disabled={!projects.length}
						className={cn("flex h-10 w-full items-center rounded-md hover:bg-sidebar-primary/50 data-placeholder:text-muted-foreground", isSidebarOpen ? "px-2" : "px-1")}
					>
						<span className="size-7 mr-2 rounded-md bg-accent text-xs font-semibold flex items-center justify-center shrink-0">
							{projects.find((p) => p.id === activeProjectId)?.name[0] ?? "P"}
						</span>
						{isSidebarOpen && (
							<>
								<span className="text-sm font-medium">
									{projects.find((p) => p.id === activeProjectId)?.name ||
										"Select Project"}
								</span>
								<ChevronUpDownIcon size={16} className="ml-auto opacity-50" />
							</>
						)}
					</motion.button>
				</AnimateIcon>
			</SelectTrigger>
			<SelectContent>
				<SelectGroup>
					<SelectLabel className="text-xs text-muted-foreground leading-2">
						Projects
					</SelectLabel>
					{projects.map((p) => (
						<SelectItem
							key={p.id}
							value={p.id.toString()}
							className="focus:bg-sidebar-primary/50 my-0.5"
							asChild
						>
							<div className="flex items-center gap-2 rounded-md px-2 py-2">
								<span className="size-6 rounded-md bg-accent text-xs font-semibold flex items-center justify-center">
									{p.name[0]}
								</span>
								<span className="text-sm font-medium">{p.name}</span>
							</div>
						</SelectItem>
					))}
				</SelectGroup>
				<SelectSeparator />
				<SelectGroup>
					<SelectItem asChild value="create_new">
						<AnimateIcon asChild animateOnHover>
							<motion.button
								className="flex w-full items-center gap-2 rounded-md px-2 py-2 text-sm font-medium hover:bg-sidebar-primary/50 focus:bg-sidebar-primary/50"
								whileHover={{ scale: 1.02 }}
								whileTap={{ scale: 0.98 }}
							>
								<PlusIcon size={14} />
								Create New Project
							</motion.button>
						</AnimateIcon>
					</SelectItem>
				</SelectGroup>
			</SelectContent>
		</Select>
	);
}
