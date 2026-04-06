import { Outlet } from "react-router-dom";
import { BackgroundLines } from "../patterns/lines";
import { Card, CardContent, CardHeader } from "../ui/card";

export function AuthLayout() {
  return (
		<div className="flex min-h-screen bg-background relative overflow-hidden">
			<BackgroundLines
				color="var(--color-primary)"
				lineCount={11}
				duration={12}
				className="bg-transparent! fixed! inset-0 pointer-events-none"
				aria-hidden="true"
			/>
			<div className="relative z-10 w-full flex items-center justify-center py-4">
				<Card>
					<CardHeader />
					<CardContent className="min-w-md max-w-xl">
						<Outlet />
					</CardContent>
				</Card>
			</div>
		</div>
	);
}
