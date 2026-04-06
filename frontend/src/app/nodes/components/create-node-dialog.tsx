import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { CheckCircle2, LoaderCircle, RefreshCw, ServerCog, Terminal } from "lucide-react";
import { CopyIcon, AnimateIcon } from "@/components/animate-ui/icons";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { useCreateNode, useNode, useRegenerateAgentToken } from "@/hooks/use-nodes.ts";
import { ApiError, NetworkError, getApiBaseUrl } from "@/lib/fetch-client.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import type { NodeResponse } from "@/types/node.ts";
import { CopyButton } from "@/components/animate-ui/components/buttons/copy";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required").max(100, "Keep the name under 100 characters"),
});

type FormValues = z.infer<typeof schema>;
type Step = 0 | 1 | 2;

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
    description: "Create the node record before Agrafa can issue a token for it.",
  },
  {
    title: "Generate token",
    description: "Fetch the raw agent token once, copy it, then move on to setup.",
  },
  {
    title: "Setup and connect",
    description: "Run the agent with the generated token and wait for the first heartbeat.",
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

function StepIndicator({ currentStep }: { currentStep: Step }) {
  return (
    <div className="grid gap-2 sm:grid-cols-3">
      {STEP_COPY.map((step, index) => {
        const isActive = currentStep === index;
        const isComplete = currentStep > index;

        return (
          <div
            key={step.title}
            className={[
              "rounded-lg border px-3 py-2 text-left transition-colors",
              isComplete
                ? "border-primary/40 bg-primary/10"
                : isActive
                  ? "border-primary/50 bg-card"
                  : "border-border bg-muted/20",
            ].join(" ")}
          >
            <div className="flex items-center gap-2">
              <span
                className={[
                  "flex h-6 w-6 items-center justify-center rounded-full border text-xs font-semibold",
                  isComplete || isActive
                    ? "border-primary text-primary"
                    : "border-muted-foreground/40 text-muted-foreground",
                ].join(" ")}
              >
                {isComplete ? <CheckCircle2 className="h-3.5 w-3.5" /> : index + 1}
              </span>
              <p className="text-sm font-medium text-foreground">{step.title}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function CodeBlock({ label, value }: { label: string; value: string }) {
  return (
		<div className="space-y-2">
			<div className="flex items-center justify-between gap-3">
				<p className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
					{label}
				</p>
				{/* <CopyButton value={value} label={label} /> */}
			</div>
			<div className="rounded-lg border border-border/70 bg-black/30 px-4 py-3 text-sm text-foreground pr-12 relative">
				<pre className="overflow-x-auto ">
					{value}
				</pre>
				<CopyButton
					content={value}
					variant={"secondary"}
					size={"sm"}
					className="absolute top-1.5 right-1.5"
				/>
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
  const apiBaseUrl = getApiBaseUrl();

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
  const installCommand = [
    `AGRAFA_AGENT_TOKEN='${rawToken ?? "<generate a token first>"}' \\`,
    `AGRAFA_API_BASE_URL='${apiBaseUrl}' \\`,
    "./agrafa-agent",
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
      toast.success(hadToken ? "New agent token generated" : "Agent token generated");
    } catch (error) {
      const message = getTokenErrorMessage(error);
      setTokenError(message);
      toast.error(message);
    }
  }

  async function handleAdvanceToSetup() {
    if (!rawToken) {
      return;
    }

    setStep(2);
    if (createdNode) {
      await nodeQuery.refetch();
    }
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
			<DialogContent className="max-w-3xl">
				<DialogHeader>
					<DialogTitle>Set up a node</DialogTitle>
					<DialogDescription>{STEP_COPY[step].description}</DialogDescription>
				</DialogHeader>

				<StepIndicator currentStep={step} />

				{step === 0 && (
					<Form {...form}>
						<form
							onSubmit={form.handleSubmit(handleCreate)}
							className="space-y-4"
						>
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
							<div className="rounded-lg border border-border/70 bg-muted/20 px-4 py-3 text-sm text-muted-foreground">
								Create the node first. Token generation and setup instructions
								stay locked until the backend confirms the node exists.
							</div>
							<div className="flex justify-end gap-2">
								<Button
									type="button"
									variant="outline"
									onClick={() => onOpenChange(false)}
								>
									Cancel
								</Button>
								<Button type="submit" disabled={createNode.isPending}>
									{createNode.isPending ? "Creating..." : "Create node"}
								</Button>
							</div>
						</form>
					</Form>
				)}

				{step === 1 && createdNode && (
					<div className="space-y-4">
						<div className="rounded-lg border border-border/70 bg-muted/20 px-4 py-3">
							<div className="flex flex-wrap items-start justify-between gap-3">
								<div className="space-y-1">
									<p className="text-sm font-medium text-foreground">
										{createdNode.name}
									</p>
									<p className="font-mono text-xs text-muted-foreground">
										{createdNode.identifier}
									</p>
								</div>
								<StatusBadge status="offline" />
							</div>
						</div>

						{tokenError && (
							<Alert variant="destructive">
								<ServerCog className="h-4 w-4" />
								<AlertTitle>Token generation failed</AlertTitle>
								<AlertDescription>{tokenError}</AlertDescription>
							</Alert>
						)}

						<Alert className="border-primary/20 bg-primary/5">
							{regenerateToken.isPending ? (
								<LoaderCircle className="h-4 w-4 animate-spin" />
							) : (
								<ServerCog className="h-4 w-4" />
							)}
							<AlertTitle>Generate the raw token</AlertTitle>
							<AlertDescription>
								Agrafa only returns the raw token when you generate it. You
								cannot continue until a token has been received in this dialog.
							</AlertDescription>
						</Alert>

						<div className="rounded-lg border border-border/70 bg-card p-4">
							<div className="flex flex-wrap items-center justify-between gap-3">
								<div>
									<p className="text-sm font-medium text-foreground">
										Agent token
									</p>
									<p className="text-sm text-muted-foreground">
										{rawToken
											? "Copy this value now. Regenerating replaces the displayed token immediately."
											: "No token generated yet."}
									</p>
								</div>
								<div className="flex gap-2">
									{/* {rawToken ? <CopyButton value={rawToken} label="Agent token" /> : null} */}
									<Button
										type="button"
										size="sm"
										variant={rawToken ? "outline" : "default"}
										onClick={() => void handleGenerateToken()}
										disabled={regenerateToken.isPending}
									>
										{regenerateToken.isPending
											? "Generating..."
											: rawToken
												? "Generate new token"
												: "Generate token"}
									</Button>
								</div>
							</div>
              <div className="mt-4 rounded-lg border border-border/70 bg-black/30 px-4 py-3 text-sm text-foreground relative">
                <pre className="overflow-x-auto">
                  {rawToken ?? "Generate the token to reveal it here."}
                  {rawToken && (
                    <CopyButton
                      variant={"secondary"}
                      content={rawToken ?? ""}
                      size={"sm"}
                      className="absolute top-1.5 right-2"
                    />
                  )}
                </pre>
              </div>
						</div>

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
								onClick={() => void handleAdvanceToSetup()}
								disabled={!rawToken || regenerateToken.isPending}
							>
								Continue to setup
							</Button>
						</div>
					</div>
				)}

				{step === 2 && createdNode && (
					<div className="space-y-4">
						<Alert>
							<Terminal className="h-4 w-4" />
							<AlertTitle>Install the agent on your server</AlertTitle>
							<AlertDescription>
								Start the `agrafa-agent` binary with the token below. Keep this
								dialog open while Agrafa waits for the first heartbeat.
							</AlertDescription>
						</Alert>

						{/* <div className="grid gap-4 md:grid-cols-2">
							<CodeBlock
								label="AGRAFA_AGENT_TOKEN"
								value={rawToken ?? "<generate a token first>"}
							/>
							<CodeBlock label="AGRAFA_API_BASE_URL" value={apiBaseUrl} />
						</div> */}

						{/* <CodeBlock label="Run command" value={installCommand} /> */}

						<div className="rounded-lg border border-border/70 bg-muted/20 px-4 py-3">
							<div className="flex flex-wrap items-start justify-between gap-3">
								<div className="space-y-1">
									<p className="text-sm font-medium text-foreground">
										Connection status
									</p>
									<p className="text-sm text-muted-foreground">
										{isNodeOnline
											? "The node is online and ready for service checks."
											: "Waiting for the first heartbeat from this node."}
									</p>
								</div>
								<StatusBadge status={liveNode?.current_state ?? "offline"} />
							</div>
							<div className="mt-4 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
								<span>
									Identifier: {liveNode?.identifier ?? createdNode.identifier}
								</span>
								<span className="hidden sm:inline">•</span>
								<span>
									Last seen:{" "}
									{liveNode?.last_seen_at
										? formatRelativeTime(liveNode.last_seen_at)
										: "Never"}
								</span>
							</div>
						</div>

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
									Start the agent with the generated token, then refresh
									manually or wait for the automatic poll.
								</AlertDescription>
							</Alert>
						)}

						<div className="flex flex-wrap justify-between gap-2">
							<div className="flex flex-wrap gap-2">
								<Button
									type="button"
									variant="outline"
									onClick={() => onOpenChange(false)}
								>
									{launchedFromService ? "Back to service" : "Close"}
								</Button>
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
									onClick={() => void handleGenerateToken()}
									disabled={regenerateToken.isPending}
								>
									{regenerateToken.isPending
										? "Generating..."
										: "Generate new token"}
								</Button>
							</div>
							<Button
								type="button"
								onClick={handleFinish}
								disabled={
									!isNodeOnline || regenerateToken.isPending || !rawToken
								}
							>
								{launchedFromService ? "Continue to service" : "Done"}
							</Button>
						</div>
					</div>
				)}
			</DialogContent>
		</Dialog>
	);
}
