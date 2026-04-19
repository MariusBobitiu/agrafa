import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { CircuitBoardIcon } from "@/components/animate-ui/icons";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Button } from "@/components/ui/button.tsx";
import { cn } from "@/lib/utils.ts";
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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { ApiError, NetworkError } from "@/lib/fetch-client.ts";
import { useNodes } from "@/hooks/use-nodes.ts";
import { useCreateService, useUpdateService } from "@/hooks/use-services.ts";
import {
  useServiceCreationStore,
  type ServiceCreationDraft,
} from "@/stores/service-creation-store.ts";
import type {
  Service,
  ServiceCreateInput,
  ServiceExecutionMode,
  ServiceUpdateInput,
} from "@/types/service.ts";

const schema = z
  .object({
    name: z.string().trim().min(1, "Name is required"),
    executionMode: z.enum(["managed", "agent"]),
    nodeId: z.string().optional(),
    checkType: z.enum(["http", "tcp"]),
    checkTarget: z.string().trim().min(1, "Target is required"),
  })
  .superRefine((values, ctx) => {
    if (values.executionMode === "agent" && !values.nodeId) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["nodeId"],
        message: "Choose the server that should run this check",
      });
    }
  });

type FormValues = z.infer<typeof schema>;

const EXECUTION_LOCATION_OPTIONS: Array<{
  value: ServiceExecutionMode;
  title: string;
  description: string;
}> = [
  {
    value: "managed",
    title: "This instance",
    description: "Runs checks from your current Agrafa server",
  },
  {
    value: "agent",
    title: "A project node",
    description: "Run checks from a node you control",
  },
];

type Props = {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onRequestNodeSetup: () => void;
  service?: Service | null;
};

const DEFAULT_FORM_VALUES: FormValues = {
  name: "",
  executionMode: "managed",
  nodeId: "",
  checkType: "http",
  checkTarget: "",
};

function getCreateServiceErrorMessage(error: unknown) {
  if (error instanceof NetworkError) {
    return error.message;
  }

  if (error instanceof ApiError) {
    if (error.status === 409) {
      return "A service with this name already exists for that location.";
    }

    if (error.status === 400 || error.status === 404) {
      return "Review the service details and try again.";
    }

    if (error.status >= 500) {
      return "Agrafa couldn't create the service right now. Try again in a moment.";
    }
  }

  return "Couldn't create the service. Try again.";
}

function getUpdateServiceErrorMessage(error: unknown) {
  if (error instanceof NetworkError) {
    return error.message;
  }

  if (error instanceof ApiError) {
    if (error.status === 409) {
      return "A service with this name already exists for that location.";
    }

    if (error.status === 400 || error.status === 404) {
      return "Review the service details and try again.";
    }

    if (error.status >= 500) {
      return "Agrafa couldn't update the service right now. Try again in a moment.";
    }
  }

  return "Couldn't update the service. Try again.";
}

export function CreateServiceDialog({
  projectId,
  open,
  onOpenChange,
  onRequestNodeSetup,
  service = null,
}: Props) {
  const pendingDraft = useServiceCreationStore((s) => s.draft);
  const pendingNodeId = useServiceCreationStore((s) => s.pendingNodeId);
  const resumeRequested = useServiceCreationStore((s) => s.resumeRequested);
  const saveDraft = useServiceCreationStore((s) => s.saveDraft);
  const consumeResume = useServiceCreationStore((s) => s.consumeResume);
  const clearDraft = useServiceCreationStore((s) => s.clear);
  const createService = useCreateService(projectId);
  const updateService = useUpdateService(projectId);
  const isEditMode = service !== null;
  const initialValues =
    isEditMode && service
      ? {
          name: service.name,
          executionMode: service.execution_mode,
          nodeId: service.node_id ? String(service.node_id) : "",
          checkType: service.check_type,
          checkTarget: service.check_target,
        }
      : resumeRequested && pendingDraft
        ? {
            ...pendingDraft,
            executionMode: (pendingNodeId
              ? "agent"
              : pendingDraft.executionMode) as ServiceExecutionMode,
            nodeId: pendingNodeId ?? pendingDraft.nodeId ?? "",
          }
        : DEFAULT_FORM_VALUES;
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: initialValues,
  });
  const executionMode = form.watch("executionMode");
  const shouldLoadNodes = open && executionMode === "agent";
  const nodesQuery = useNodes(projectId, { enabled: shouldLoadNodes });
  const agentNodes = nodesQuery.data?.nodes ?? [];
  const hasNoAgentNodes =
    executionMode === "agent" && nodesQuery.isSuccess && agentNodes.length === 0;
  const isAgentSelectionUnavailable =
    executionMode === "agent" && (nodesQuery.isLoading || nodesQuery.isError || hasNoAgentNodes);

  useEffect(() => {
    if (open && isEditMode && service) {
      form.reset({
        name: service.name,
        executionMode: service.execution_mode,
        nodeId: service.node_id ? String(service.node_id) : "",
        checkType: service.check_type,
        checkTarget: service.check_target,
      });
      return;
    }

    if (open && resumeRequested && pendingDraft) {
      form.reset({
        ...pendingDraft,
        executionMode: (pendingNodeId
          ? "agent"
          : pendingDraft.executionMode) as ServiceExecutionMode,
        nodeId: pendingNodeId ?? pendingDraft.nodeId ?? "",
      });
      consumeResume();
    }
  }, [
    consumeResume,
    form,
    isEditMode,
    open,
    pendingDraft,
    pendingNodeId,
    resumeRequested,
    service,
  ]);

  function handleOpenChange(nextOpen: boolean, options?: { preserveDraft?: boolean }) {
    onOpenChange(nextOpen);
    if (!nextOpen) {
      form.reset(
        isEditMode && service
          ? {
              name: service.name,
              executionMode: service.execution_mode,
              nodeId: service.node_id ? String(service.node_id) : "",
              checkType: service.check_type,
              checkTarget: service.check_target,
            }
          : DEFAULT_FORM_VALUES,
      );

      if (!isEditMode && !options?.preserveDraft) {
        clearDraft();
      }
    }
  }

  async function onSubmit(values: FormValues) {
    if (isEditMode && service) {
      const payload: ServiceUpdateInput = {
        name: values.name.trim(),
        check_type: values.checkType,
        check_target: values.checkTarget.trim(),
      };

      try {
        await updateService.mutateAsync({ id: service.id, payload });
        toast.success("Service updated");
        handleOpenChange(false);
      } catch (error) {
        toast.error(getUpdateServiceErrorMessage(error));
      }

      return;
    }

    const payload: ServiceCreateInput = {
      project_id: projectId,
      execution_mode: values.executionMode,
      name: values.name.trim(),
      check_type: values.checkType,
      check_target: values.checkTarget.trim(),
    };

    if (values.executionMode === "agent" && values.nodeId) {
      payload.node_id = Number(values.nodeId);
    }

    try {
      await createService.mutateAsync(payload);
      toast.success("Service created");
      clearDraft();
      handleOpenChange(false);
    } catch (error) {
      toast.error(getCreateServiceErrorMessage(error));
    }
  }

  function handleNodeSetupClick() {
    const draft: ServiceCreationDraft = {
      name: form.getValues("name"),
      executionMode: form.getValues("executionMode"),
      nodeId: form.getValues("nodeId") ?? "",
      checkType: form.getValues("checkType"),
      checkTarget: form.getValues("checkTarget"),
    };

    saveDraft(draft);
    handleOpenChange(false, { preserveDraft: true });
    onRequestNodeSetup();
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => handleOpenChange(nextOpen)}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditMode ? "Edit service" : "Add service"}</DialogTitle>
          <DialogDescription>
            {isEditMode
              ? "Update the service definition used by the current health check."
              : "Choose what to monitor and where the checks should run from."}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g. API health" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {!isEditMode && (
              <FormField
                control={form.control}
                name="executionMode"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Where should checks run from?</FormLabel>
                    <FormDescription>
                      Choose whether Agrafa runs the check itself or through a node you manage.
                    </FormDescription>
                    <div className="grid gap-3 sm:grid-cols-2">
                      {EXECUTION_LOCATION_OPTIONS.map((option) => {
                        const isSelected = field.value === option.value;

                        return (
                          <label
                            key={option.value}
                            className={cn(
                              "flex cursor-pointer rounded-lg border px-4 py-3 transition-colors",
                              isSelected
                                ? "border-primary bg-primary/10 shadow-[0_0_0_1px_hsl(var(--primary))]"
                                : "border-border bg-card hover:border-primary/40 hover:bg-muted/30",
                            )}
                          >
                            <input
                              type="radio"
                              name={field.name}
                              value={option.value}
                              checked={isSelected}
                              className="sr-only"
                              onChange={(event) => {
                                field.onChange(event.target.value);
                                if (event.target.value === "managed") {
                                  form.clearErrors("nodeId");
                                }
                              }}
                            />
                            <span className="flex w-full items-start justify-between gap-3">
                              <span className="space-y-1">
                                <span className="block text-sm font-medium text-foreground">
                                  {option.title}
                                </span>
                                <span className="block text-sm text-muted-foreground">
                                  {option.description}
                                </span>
                              </span>
                              <span
                                className={cn(
                                  "mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border",
                                  isSelected ? "border-primary" : "border-muted-foreground/40",
                                )}
                              >
                                <span
                                  className={cn(
                                    "h-2 w-2 rounded-full bg-primary transition-opacity",
                                    !isSelected && "opacity-0",
                                  )}
                                />
                              </span>
                            </span>
                          </label>
                        );
                      })}
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            {!isEditMode && executionMode === "agent" && (
              <FormField
                control={form.control}
                name="nodeId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Select node</FormLabel>
                    <FormDescription>
                      Pick the node that should run this check for this project.
                    </FormDescription>
                    {nodesQuery.isLoading ? (
                      <div className="rounded-lg border border-dashed border-border bg-muted/20 px-4 py-3 text-sm text-muted-foreground">
                        Loading your available nodes...
                      </div>
                    ) : nodesQuery.isError ? (
                      <Alert variant="destructive">
                        <CircuitBoardIcon size={16} />
                        <AlertTitle>Couldn't load your nodes</AlertTitle>
                        <AlertDescription className="mt-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                          <span>Try again to choose where this check should run.</span>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => void nodesQuery.refetch()}
                          >
                            Retry
                          </Button>
                        </AlertDescription>
                      </Alert>
                    ) : hasNoAgentNodes ? (
                      <Alert>
                        <CircuitBoardIcon size={16} />
                        <AlertTitle>No nodes available yet</AlertTitle>
                        <AlertDescription className="mt-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                          <span>
                            Set up a node first so Agrafa has somewhere to run this check from.
                          </span>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={handleNodeSetupClick}
                          >
                            Set up a node
                          </Button>
                        </AlertDescription>
                      </Alert>
                    ) : (
                      <>
                        <Select onValueChange={field.onChange} value={field.value}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Select a node" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {agentNodes.map((node) => (
                              <SelectItem key={node.id} value={node.id.toString()}>
                                {node.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormDescription>
                          {agentNodes.length} available {agentNodes.length === 1 ? "node" : "nodes"}{" "}
                          in this project
                        </FormDescription>
                      </>
                    )}
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name="checkType"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Check type</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="http">HTTP</SelectItem>
                      <SelectItem value="tcp">TCP</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="checkTarget"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Target or endpoint</FormLabel>
                  <FormControl>
                    <Input placeholder="https://api.example.com/health" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" type="button" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={
                  (isEditMode ? updateService.isPending : createService.isPending) ||
                  (!isEditMode && isAgentSelectionUnavailable)
                }
              >
                {isEditMode
                  ? updateService.isPending
                    ? "Saving..."
                    : "Save changes"
                  : createService.isPending
                    ? "Creating..."
                    : "Create"}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
