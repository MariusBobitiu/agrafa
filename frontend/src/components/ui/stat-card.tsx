import { Card, CardContent } from "./card.tsx";
import { Skeleton } from "./skeleton.tsx";
import { AnimateIcon, type IconProps } from "../animate-ui/icons/icon.tsx";
import type { JSX } from "react/jsx-runtime";

type StatCardProps = {
  title: string;
  value: string | number;
  icon: (props: IconProps<any>) => JSX.Element;
  description?: string;
  loading?: boolean;
};

export function StatCard({ title, value, icon: Icon, description, loading }: StatCardProps) {
  return (
		<AnimateIcon asChild animateOnHover>
			<Card>
				<CardContent className="p-5">
					<div className="flex items-center justify-between">
						<p className="text-sm text-muted-foreground">{title}</p>
						<Icon size={20} className="text-muted-foreground" />
					</div>
					{loading ? (
						<Skeleton className="mt-2 h-7 w-24" />
					) : (
						<p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
					)}
					{description && (
						<p className="mt-1 text-xs text-muted-foreground">{description}</p>
					)}
				</CardContent>
			</Card>
		</AnimateIcon>
	);
}
