import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import {
	CheckCircle2,
	CopyIcon,
	LoaderCircle,
	RefreshCw,
	ServerCog,
} from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog.tsx";
import {
	Form,
	FormControl,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import {
	useCreateNode,
	useNode,
	useRegenerateAgentToken,
} from "@/hooks/use-nodes.ts";
import {
	ApiError,
	NetworkError,
	getAgentApiBaseUrl,
} from "@/lib/fetch-client.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import type { NodeResponse } from "@/types/node.ts";
import { CopyButton } from "@/components/animate-ui/components/buttons/copy";
import { AnimatePresence, motion } from "motion/react";

const schema = z.object({
	name: z
		.string()
		.trim()
		.min(1, "Name is required")
		.max(100, "Keep the name under 100 characters"),
});

type FormValues = z.infer<typeof schema>;
type Step = 0 | 1 | 2;

const AGENT_IMAGE = "ghcr.io/mariusbobitiu/agrafa-agent:latest";

type Props = {
	projectId: number;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	launchedFromService?: boolean;
	onComplete?: (nodeId: number) => void;
};

const STEP_COPY = [
	{
		title: "Create node",
		description:
			"Create the node record before Agrafa can issue a token for it.",
	},
	{
		title: "Install agent",
		description:
			"Run the agent container on the target machine using the generated token.",
	},
	{
		title: "Wait for heartbeat",
		description:
			"Agrafa will mark the node online as soon as it receives the first heartbeat.",
	},
] as const;

function getNodeCreateErrorMessage(error: unknown) {
	if (error instanceof NetworkError) {
		return error.message;
	}

	if (error instanceof ApiError) {
		if (error.status === 409) {
			return "A node with that name already exists in this project.";
		}

		if (error.status === 400 || error.status === 404) {
			return "Review the node details and try again.";
		}

		if (error.status >= 500) {
			return "Agrafa couldn't create the node right now. Try again in a moment.";
		}
	}

	return "Couldn't create the node. Try again.";
}

function getTokenErrorMessage(error: unknown) {
	if (error instanceof NetworkError) {
		return error.message;
	}

	if (error instanceof ApiError) {
		if (error.status === 404) {
			return "This node is no longer available. Create a new one and try again.";
		}

		if (error.status >= 500) {
			return "Agrafa couldn't generate a new token right now. Try again in a moment.";
		}
	}

	return "Couldn't generate a token. Try again.";
}

function getNodeConnectionErrorMessage(error: unknown) {
	if (error instanceof NetworkError) {
		return error.message;
	}

	if (error instanceof ApiError && error.status === 404) {
		return "This node could not be found anymore.";
	}

	return "Agrafa couldn't refresh this node right now.";
}

function CodeBlock({ label, value }: { label?: string; value: string }) {
	return (
		<div className="flex-1 space-y-2">
			{label && (
				<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground">
					{label}
				</p>
			)}
			<div className="relative min-w-0 overflow-hidden rounded-xl border border-border/70 bg-muted/20">
				<pre className="max-h-48 overflow-auto px-4 py-3 pr-14 text-sm leading-6 text-foreground">
					{value}
				</pre>
				<CopyButton
					content={value}
					variant={"secondary"}
					size={"sm"}
					className="absolute right-2 top-2"
				/>
			</div>
		</div>
	);
}

function SectionTitle({
	title,
	description,
}: {
	title: string;
	description: string;
}) {
	return (
		<div className="space-y-1">
			<h3 className="text-base font-semibold text-foreground">{title}</h3>
			<p className="text-sm text-muted-foreground">{description}</p>
		</div>
	);
}

function NodeSnapshot({
	node,
	status,
	lastSeenAt,
}: {
	node: NodeResponse;
	status: "online" | "offline" | "unknown";
	lastSeenAt?: string | null;
}) {
	return (
		<div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/15 px-4 py-3">
			<div className="w-full space-y-1">
				<div className="w-full flex items-center justify-between gap-2">
					<p className="text-sm font-medium text-foreground">{node.name}</p>
					<StatusBadge status={status} />
				</div>
				<div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
					{lastSeenAt ? (
						<>
							<span className="font-mono">{node.identifier}</span>
							<span className="hidden sm:inline">•</span>
							<span>Last seen {formatRelativeTime(lastSeenAt)}</span>
						</>
					) : null}
				</div>
			</div>
		</div>
	);
}

export function CreateNodeDialog({
	projectId,
	open,
	onOpenChange,
	launchedFromService = false,
	onComplete,
}: Props) {
	const queryClient = useQueryClient();
	const createNode = useCreateNode(projectId);
	const regenerateToken = useRegenerateAgentToken();
	const [step, setStep] = useState<Step>(0);
	const [createdNode, setCreatedNode] = useState<NodeResponse | null>(null);
	const [rawToken, setRawToken] = useState<string | null>(null);
	const [tokenError, setTokenError] = useState<string | null>(null);
	const agentApiBaseUrl = getAgentApiBaseUrl();
  const [regenerated, setRegenerated] = useState(false);

	const form = useForm<FormValues>({
		resolver: zodResolver(schema),
		defaultValues: { name: "" },
	});

	const nodeQuery = useNode(createdNode?.id ?? 0, {
		enabled: open && step === 2 && !!createdNode,
		refetchInterval: (query) =>
			open && step === 2 && createdNode
				? query.state.data?.node.current_state === "online"
					? false
					: 4_000
				: false,
	});

	const liveNode = nodeQuery.data?.node;
	const isNodeOnline = liveNode?.current_state === "online";
	// const pullCommand = `docker pull ${AGENT_IMAGE}`;
	const runCommand = [
		"docker run -d \\",
		`  --restart unless-stopped \\`,
		`  --name agrafa-agent-${createdNode?.id ?? "node"} \\`,
		"  --pid=host \\",
		`  -e AGRAFA_API_BASE_URL='${agentApiBaseUrl}' \\`,
		`  -e AGRAFA_AGENT_TOKEN='${rawToken ?? "<generate a token first>"}' \\`,
		// `  -e AGRAFA_NODE_ID='${createdNode?.id ?? "<node id>"}' \\`,
		"  -e HOST_PROC=/host/proc \\",
		"  -e HOST_SYS=/host/sys \\",
		"  -e HOST_ETC=/host/etc \\",
		"  -e HOST_ROOT=/host \\",
		"  -e AGRAFA_DISK_PATH=/host \\",
		"  -v /proc:/host/proc:ro \\",
		"  -v /sys:/host/sys:ro \\",
		"  -v /etc:/host/etc:ro \\",
		"  -v /:/host:ro \\",
		`  ${AGENT_IMAGE}`,
	].join("\n");

	useEffect(() => {
		if (!open) {
			setStep(0);
			setCreatedNode(null);
			setRawToken(null);
			setTokenError(null);
			form.reset({ name: "" });
		}
	}, [form, open]);

	useEffect(() => {
		if (!open || !isNodeOnline) {
			return;
		}

		void queryClient.invalidateQueries({ queryKey: ["nodes", projectId] });
		void queryClient.invalidateQueries({ queryKey: ["overview", projectId] });
	}, [isNodeOnline, open, projectId, queryClient]);

	async function handleCreate(values: FormValues) {
		try {
			const response = await createNode.mutateAsync({
				project_id: projectId,
				name: values.name.trim(),
			});

			setCreatedNode(response.node);
			setRawToken(null);
			setTokenError(null);

			// Generate the token immediately after creating the node so it's ready by the time the user advances to the next step. This also surfaces token generation errors earlier in the flow.
			try {
				const tokenResponse = await regenerateToken.mutateAsync(
					response.node.id,
				);
				setRawToken(tokenResponse.agent_token);
				toast.success("Agent token generated");
			} catch (error) {
				const message = getTokenErrorMessage(error);
				setTokenError(message);
				toast.error(message);
			}
			setStep(1);
			toast.success("Node created");
		} catch (error) {
			toast.error(getNodeCreateErrorMessage(error));
		}
	}

	async function handleGenerateToken() {
		if (!createdNode) {
			return;
		}

		try {
			const hadToken = rawToken !== null;
			setTokenError(null);
			setRawToken(null);
			const response = await regenerateToken.mutateAsync(createdNode.id);
			setRawToken(response.agent_token);
			toast.success(
				hadToken ? "New agent token generated" : "Agent token generated",
			);
		} catch (error) {
			const message = getTokenErrorMessage(error);
			setTokenError(message);
			toast.error(message);
		}
	}

	function handleAdvanceToHeartbeat() {
		setStep(2);
	}

	function handleFinish() {
		if (!createdNode || !isNodeOnline) {
			return;
		}

		onComplete?.(createdNode.id);
		if (!onComplete) {
			onOpenChange(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-3xl max-h-[85vh] flex flex-col overflow-hidden border-border/70 bg-background p-0">
				<DialogHeader className="border-b border-border/70 bg-muted/10 px-6 py-5">
					<div className="space-y-1">
						<div className="flex items-center justify-between gap-3">
							<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground mb-2">
								Step {step + 1} of {STEP_COPY.length}
							</p>
						</div>
						<DialogTitle>Install and start the agent</DialogTitle>
						<DialogDescription className="max-w-2xl">
							{STEP_COPY[step].description}
						</DialogDescription>
					</div>
				</DialogHeader>
				<div className="flex-1 space-y-3 overflow-x-hidden overflow-y-auto px-6 pb-2">
					{step === 0 && (
						<Form {...form}>
							<form
								onSubmit={form.handleSubmit(handleCreate)}
								className="space-y-6"
								id="create-node-form"
							>
								<SectionTitle
									title="Create the node record"
									description="This registers the machine in Agrafa so a one-time agent token can be issued for it."
								/>
								<FormField
									control={form.control}
									name="name"
									render={({ field }) => (
										<FormItem>
											<FormLabel>Name</FormLabel>
											<FormControl>
												<Input placeholder="e.g. fra1-web-01" {...field} />
											</FormControl>
											<FormMessage />
										</FormItem>
									)}
								/>
								<div className="rounded-xl border border-dashed border-border/70 bg-muted/15 px-4 py-3 text-sm text-muted-foreground">
									Create the node first. Token generation and setup instructions
									stay locked until the backend confirms the node exists.
								</div>
							</form>
						</Form>
					)}

					{step === 1 && createNode && (
						<div className="space-y-6">
							<SectionTitle
								title="Install and start the agent"
								description="Run these commands on the target machine. Once the container is up, move to heartbeat detection."
							/>
							<div className="grid gap-4">
								{/* <CodeBlock label="Pull image" value={pullCommand} /> */}
								<CodeBlock label="Run this command" value={runCommand} />
							</div>

							<div className="rounded-xl border border-dashed border-border/70 bg-muted/15 px-4 py-3 text-sm text-muted-foreground">
								This Linux example mounts host system paths so the agent can
								report host metrics. Add{" "}
								<code className="mx-1 rounded bg-background px-1 py-0.5">
									--network host
								</code>
								if you plan to monitor services bound to{" "}
								<code className="mx-1 rounded bg-background px-1 py-0.5">
									localhost
								</code>
								.
							</div>

							{tokenError && (
								<Alert variant="destructive">
									<ServerCog className="h-4 w-4" />
									<AlertTitle>Token generation failed</AlertTitle>
									<AlertDescription>{tokenError}</AlertDescription>
								</Alert>
							)}

							<div className="space-y-2">
								<div className="flex items-center">
									<span className="size-3 rounded-full bg-primary animate-pulse mr-2" />
									<p className="text-sm text-foreground">
										After running this, come back here
									</p>
								</div>
								<p className="text-sm text-muted-foreground">
									We&apos;ll automatically detect when your node is online.
								</p>
							</div>

							<div className="space-y-2">
								<p className="text-sm text-foreground">Having trouble?</p>
								<p className="text-sm text-muted-foreground">
									Make sure the container is running and can reach the internet.
									If it still doesn&apos;t connect, regenerate the token and
									restart the agent.
								</p>
								<div className="flex flex-wrap gap-2">
									<Button
										variant="outline"
										size="sm"
										onClick={() =>
											void regenerateToken
												.mutateAsync(createdNode!.id)
												.then((response) => {
													setRawToken(response.agent_token);
													toast.success("New agent token generated");
													setRegenerated(true);
												})
												.catch((error) => {
													const message = getTokenErrorMessage(error);
													setTokenError(message);
													toast.error(message);
												})
										}
										disabled={regenerateToken.isPending}
										aria-label="Regenerate token"
										aria-disabled={regenerateToken.isPending}
										className="hover:bg-secondary hover:text-secondary-foreground"
									>
										{regenerateToken.isPending
											? "Generating new token..."
											: "Regenerate token"}
									</Button>
									<AnimatePresence mode="wait">
										{rawToken && regenerated && (
											<motion.div
												key="new-token"
												initial={{ opacity: 0, scale: 0.98 }}
												animate={{ opacity: 1, scale: 1 }}
												exit={{ opacity: 0, scale: 0.98 }}
												transition={{ duration: 0.2 }}
												className="flex-1 relative rounded-md border border-border/70 bg-muted/20 px-2 py-1 pr-12 text-sm text-muted-foreground"
											>
												<pre className="max-h-48 overflow-auto text-sm leading-7 text-muted-foreground font-mono">
													{rawToken}
												</pre>
												<CopyIcon
													className="h-3.5 w-3.5 absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground cursor-pointer"
													onClick={() => {
														navigator.clipboard.writeText(rawToken);
														toast.success("Token copied to clipboard");
													}}
												/>
											</motion.div>
										)}
									</AnimatePresence>
								</div>
							</div>
						</div>
					)}
					{step === 2 && createdNode && (
						<div className="space-y-6">
							<SectionTitle
								title="Wait for the first heartbeat"
								description="Once the container is running, Agrafa will mark this node online as soon as the first pulse arrives."
							/>

							{createdNode ? (
								<div className="pt-1">
                  <p className="text-sm text-foreground mb-2">
                    Node status
                  </p>
									<NodeSnapshot
										node={createdNode}
										status={liveNode?.current_state ?? "offline"}
										lastSeenAt={liveNode?.last_seen_at}
									/>
								</div>
							) : null}

							{nodeQuery.isError ? (
								<Alert variant="destructive">
									<ServerCog className="h-4 w-4" />
									<AlertTitle>Couldn't refresh node status</AlertTitle>
									<AlertDescription>
										{getNodeConnectionErrorMessage(nodeQuery.error)}
									</AlertDescription>
								</Alert>
							) : isNodeOnline ? (
								<Alert className="border-primary/20 bg-primary/5">
									<CheckCircle2 className="h-4 w-4 text-primary" />
									<AlertTitle>Node connected</AlertTitle>
									<AlertDescription>
										Agrafa received a heartbeat. You can finish setup now.
									</AlertDescription>
								</Alert>
							) : (
								<Alert className="border-primary/20 bg-primary/5">
									<LoaderCircle className="h-4 w-4 animate-spin text-primary" />
									<AlertTitle>Waiting for connection</AlertTitle>
									<AlertDescription>
										If the container is already running, refresh manually or
										wait for the next automatic poll.
									</AlertDescription>
								</Alert>
							)}
						</div>
					)}
				</div>
				<DialogFooter className="border-t border-border/70 bg-muted/10 px-6 py-5">
					{/* Buttons */}
					{step === 0 && (
						<div className="flex justify-end gap-2">
							<Button
								type="button"
								variant="outline"
								onClick={() => onOpenChange(false)}
							>
								Cancel
							</Button>
							<Button
								type="submit"
								form="create-node-form"
								disabled={createNode.isPending}
							>
								{createNode.isPending ? "Creating..." : "Create node"}
							</Button>
						</div>
					)}
					{step === 1 && (
						<div className="flex justify-between gap-2">
							<Button
								type="button"
								variant="outline"
								onClick={() => onOpenChange(false)}
							>
								{launchedFromService ? "Back to service" : "Close"}
							</Button>
							<Button
								type="button"
								className="min-w-32"
								onClick={handleAdvanceToHeartbeat}
							>
								Continue
							</Button>
						</div>
					)}
					{step === 2 && (
						<div className="w-full flex flex-wrap justify-between gap-2">
							<div className="flex flex-wrap gap-2">
								<Button
									type="button"
									variant="outline"
									onClick={() => void nodeQuery.refetch()}
								>
									<RefreshCw className="mr-1.5 h-3.5 w-3.5" />
									Refresh
								</Button>
								<Button
									type="button"
									variant="outline"
									onClick={() => {
										// Allow regenerating the token in case the agent fails to connect with the current one. This can help recover from token leaks or mistakes without having to create a new node.
										void handleGenerateToken();
										setStep(1); // Go back to the token generation step to show the new token for copying.
									}}
									disabled={regenerateToken.isPending}
								>
									{regenerateToken.isPending
										? "Generating..."
										: "Regenerate token"}
								</Button>
							</div>
							<Button
								type="button"
								onClick={handleFinish}
								disabled={
									!isNodeOnline || regenerateToken.isPending || !rawToken
								}
                className="min-w-32"
							>
								{launchedFromService ? "Continue to service" : "Done"}
							</Button>
						</div>
					)}
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
