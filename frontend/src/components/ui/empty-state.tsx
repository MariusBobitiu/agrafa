import type { JSX, ReactNode } from "react";
import { Button } from "./button.tsx";
import { AnimateIcon, type IconProps } from "../animate-ui/icons/icon.tsx";

type EmptyStateProps = {
  icon?: (props: IconProps<any>) => JSX.Element;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
  children?: ReactNode;
};

export function EmptyState({ icon: Icon, title, description, action, children }: EmptyStateProps) {
  return (
		<div className="flex flex-col items-center justify-center py-32 text-center border border-muted rounded-xl">
			{Icon && (
				<AnimateIcon asChild animateOnHover>
					<div className="mb-4 rounded-full bg-muted/50 p-4">
						<Icon size={24} className="text-muted-foreground" />
					</div>
				</AnimateIcon>
			)}
			<h3 className="text-sm font-semibold text-foreground">{title}</h3>
			{description && (
				<p className="mt-1 text-sm text-muted-foreground max-w-xs">
					{description}
				</p>
			)}
			{action && (
				<Button size="sm" className="mt-4" onClick={action.onClick}>
					{action.label}
				</Button>
			)}
			{children}
		</div>
	);
}
