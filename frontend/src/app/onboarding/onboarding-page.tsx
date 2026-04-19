import { zodResolver } from "@hookform/resolvers/zod";
import { AnimatePresence, motion } from "framer-motion";
import { PlusIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { z } from "zod";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card.tsx";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Progress } from "@/components/ui/progress.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { authApi } from "@/data/auth.ts";
import { projectInvitationsApi } from "@/data/project-invitations.ts";
import { useAuth } from "@/hooks/use-auth.ts";
import { useInstanceSettings } from "@/hooks/use-instance-settings.ts";
import { useCreateProject, useProjects, useUpdateProject } from "@/hooks/use-projects.ts";
import { isEmailDeliveryAvailableOnInstance } from "@/lib/instance-settings.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import type {
  ProjectInvitationCreateResult,
  ProjectInvitationRole,
} from "@/types/project-invitation.ts";
import type { Project } from "@/types/project.ts";
import { useMeta } from "@/hooks/use-meta";

const projectSchema = z.object({
  name: z.string().trim().min(1, "Project name is required").max(100),
});

const inviteSchema = z.object({
  invitations: z
    .array(
      z.object({
        email: z.string().trim().email("Enter a valid email"),
        role: z.enum(["admin", "viewer"]),
      }),
    )
    .superRefine((value, ctx) => {
      const seen = new Map<string, number>();

      value.forEach((invite, index) => {
        const normalized = invite.email.trim().toLowerCase();
        if (!normalized) return;

        const existingIndex = seen.get(normalized);
        if (existingIndex !== undefined) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ["invitations", index, "email"],
            message: "This email is already in the list",
          });
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ["invitations", existingIndex, "email"],
            message: "This email is already in the list",
          });
          return;
        }

        seen.set(normalized, index);
      });
    }),
});

type ProjectFormValues = z.infer<typeof projectSchema>;
type InviteFormValues = z.infer<typeof inviteSchema>;

const STEP_COPY = {
  project: {
    title: "Create your first project",
    description: "Projects group your nodes, services, and alerts.",
  },
  invite: {
    title: "Invite your team",
    description: "Add teammates now or skip and invite them later from the app.",
  },
  finish: {
    title: "Finish setup",
    description: "Review what was created, complete onboarding, and continue to your overview.",
  },
} as const;

type Step = keyof typeof STEP_COPY;

function getInviteResultLabel(result: ProjectInvitationCreateResult) {
  if (result.status === "created") return "Invited";
  if (result.error_code === "already_invited") return "Already invited";
  if (result.error_code === "duplicate_in_request") return "Duplicate in request";
  if (result.error_code === "invalid_email") return "Invalid email";
  if (result.error_code === "invalid_role") return "Invalid role";
  return "Failed";
}

function getInviteResultVariant(result: ProjectInvitationCreateResult) {
  return result.status === "created" ? "default" : "secondary";
}

export function OnboardingPage() {
  useMeta({
    title: "Get Started with Agrafa",
    description: "Set up your first project and invite your team to start monitoring with Agrafa",
  });
  const { user, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>("project");
  const [project, setProject] = useState<Project | null>(null);
  const [inviteResults, setInviteResults] = useState<ProjectInvitationCreateResult[]>([]);
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const { data: projectsData, isLoading: isProjectsLoading } = useProjects();
  const { data: instanceSettingsData } = useInstanceSettings();
  const createProject = useCreateProject();
  const updateProject = useUpdateProject();
  const setActiveProjectId = useUIStore((s) => s.setActiveProjectId);
  const isEmailDeliveryAvailable = isEmailDeliveryAvailableOnInstance(
    instanceSettingsData?.settings ?? [],
  );
  const visibleSteps: Step[] = isEmailDeliveryAvailable
    ? ["project", "invite", "finish"]
    : ["project", "finish"];
  const currentStepIndex = visibleSteps.indexOf(step);
  const safeStepIndex = currentStepIndex >= 0 ? currentStepIndex : 0;
  const currentStep = STEP_COPY[step];
  const totalSteps = visibleSteps.length;

  useEffect(() => {
    if (step === "invite" && !isEmailDeliveryAvailable) {
      setStep("finish");
    }
  }, [isEmailDeliveryAvailable, step]);

  const projectForm = useForm<ProjectFormValues>({
    resolver: zodResolver(projectSchema),
    defaultValues: { name: "" },
  });

  const inviteForm = useForm<InviteFormValues>({
    resolver: zodResolver(inviteSchema),
    defaultValues: {
      invitations: [{ email: "", role: "viewer" satisfies ProjectInvitationRole }],
    },
  });

  const inviteFields = useFieldArray({
    control: inviteForm.control,
    name: "invitations",
  });

  const inviteMutation = useMutation({
    mutationFn: (values: InviteFormValues) => {
      if (!project) throw new Error("Project missing");

      return projectInvitationsApi.createBatch({
        project_id: project.id,
        invitations: values.invitations.map((invite) => ({
          email: invite.email.trim(),
          role: invite.role,
        })),
      });
    },
  });

  const completeOnboarding = useMutation({
    mutationFn: async () => {
      await authApi.completeOnboarding();
      await refreshUser();
    },
    onSuccess: () => {
      toast.success("Onboarding completed");
      navigate("/overview", { replace: true });
    },
  });

  useEffect(() => {
    if (project || !projectsData?.projects.length) return;

    const existingProject =
      projectsData.projects.find((item) => item.id === activeProjectId) ??
      projectsData.projects[0] ??
      null;

    if (!existingProject) return;

    setProject(existingProject);
    setActiveProjectId(existingProject.id);
    projectForm.reset({ name: existingProject.name });
  }, [activeProjectId, project, projectForm, projectsData, setActiveProjectId]);

  async function handleCreateProject(values: ProjectFormValues) {
    const trimmedName = values.name.trim();
    const result = project
      ? await updateProject.mutateAsync({ id: project.id, payload: { name: trimmedName } })
      : await createProject.mutateAsync({ name: trimmedName });

    setProject(result.project);
    setActiveProjectId(result.project.id);
    setStep(isEmailDeliveryAvailable ? "invite" : "finish");
  }

  async function handleInviteSubmit(values: InviteFormValues) {
    const response = await inviteMutation.mutateAsync(values);
    setInviteResults(response.results);
    setStep("finish");
  }

  function handleSkipInvites() {
    setInviteResults([]);
    setStep("finish");
  }

  async function handleCompleteOnboarding() {
    await completeOnboarding.mutateAsync();
  }

  const createdInviteCount = inviteResults.filter((result) => result.status === "created").length;
  const failedInviteCount = inviteResults.length - createdInviteCount;
  const hasExistingProject = (projectsData?.projects.length ?? 0) > 0;
  const isResolvingProject = isProjectsLoading && !project;
  const stepTitle =
    step === "project"
      ? hasExistingProject
        ? "Name your project"
        : `Welcome, ${user?.name.split(" ")[0]}`
      : currentStep.title;

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Card className="w-full max-w-3xl">
        <CardHeader className="space-y-4">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">
              Step {safeStepIndex + 1} of {totalSteps}
            </p>
            <CardTitle className="text-2xl">{stepTitle}</CardTitle>
            <p className="text-sm text-muted-foreground">
              {step === "project"
                ? hasExistingProject
                  ? "Choose the name for your initial project and get Agrafa ready for your team."
                  : "Set up your first project and get Agrafa ready for your team."
                : currentStep.description}
            </p>
          </div>
          <Progress value={((safeStepIndex + 1) / totalSteps) * 100} className="h-2" />
        </CardHeader>
        <CardContent className="space-y-6">
          <AnimatePresence mode="wait">
            <motion.div
              key={step}
              initial={{ opacity: 0, x: 16 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -16 }}
              transition={{ duration: 0.15 }}
            >
              {step === "project" && (
                <div className="space-y-6">
                  <Form {...projectForm}>
                    <form
                      onSubmit={projectForm.handleSubmit(handleCreateProject)}
                      className="space-y-4"
                    >
                      <FormField
                        control={projectForm.control}
                        name="name"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Project name</FormLabel>
                            <FormControl>
                              <Input placeholder="e.g. Production" {...field} />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      {hasExistingProject && (
                        <Alert>
                          <AlertTitle>Initial project ready</AlertTitle>
                          <AlertDescription>
                            The backend already provisioned your first project. This step updates
                            its name instead of creating another one.
                          </AlertDescription>
                        </Alert>
                      )}

                      <div className="flex justify-end">
                        <Button
                          type="submit"
                          disabled={
                            isResolvingProject || createProject.isPending || updateProject.isPending
                          }
                        >
                          {isResolvingProject
                            ? "Preparing..."
                            : createProject.isPending || updateProject.isPending
                              ? hasExistingProject
                                ? "Saving..."
                                : "Creating..."
                              : "Continue"}
                        </Button>
                      </div>
                    </form>
                  </Form>
                </div>
              )}

              {step === "invite" && (
                <div className="space-y-6">
                  <Alert>
                    <AlertTitle>{project?.name}</AlertTitle>
                    <AlertDescription>
                      Your project has been created. Invite admins or viewers now, or skip and do it
                      later from your workspace.
                    </AlertDescription>
                  </Alert>

                  <Form {...inviteForm}>
                    <form
                      onSubmit={inviteForm.handleSubmit(handleInviteSubmit)}
                      className="space-y-4"
                    >
                      <div className="space-y-3">
                        {inviteFields.fields.map((field, index) => (
                          <div
                            key={field.id}
                            className="grid gap-3 rounded-lg border border-border p-4 md:grid-cols-[1fr_180px_auto]"
                          >
                            <FormField
                              control={inviteForm.control}
                              name={`invitations.${index}.email`}
                              render={({ field: emailField }) => (
                                <FormItem>
                                  <FormLabel>Email</FormLabel>
                                  <FormControl>
                                    <Input placeholder="teammate@example.com" {...emailField} />
                                  </FormControl>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />

                            <FormField
                              control={inviteForm.control}
                              name={`invitations.${index}.role`}
                              render={({ field: roleField }) => (
                                <FormItem>
                                  <FormLabel>Role</FormLabel>
                                  <Select
                                    value={roleField.value}
                                    onValueChange={roleField.onChange}
                                  >
                                    <FormControl>
                                      <SelectTrigger>
                                        <SelectValue />
                                      </SelectTrigger>
                                    </FormControl>
                                    <SelectContent>
                                      <SelectItem value="viewer">Viewer</SelectItem>
                                      <SelectItem value="admin">Admin</SelectItem>
                                    </SelectContent>
                                  </Select>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />

                            <div className="flex items-end">
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                disabled={inviteFields.fields.length === 1}
                                onClick={() => inviteFields.remove(index)}
                              >
                                <XIcon />
                              </Button>
                            </div>
                          </div>
                        ))}
                      </div>

                      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => inviteFields.append({ email: "", role: "viewer" })}
                        >
                          <PlusIcon />
                          Add another invite
                        </Button>

                        <div className="flex gap-2">
                          <Button type="button" variant="ghost" onClick={handleSkipInvites}>
                            Skip for now
                          </Button>
                          <Button type="submit" disabled={inviteMutation.isPending}>
                            {inviteMutation.isPending ? "Sending..." : "Send invites"}
                          </Button>
                        </div>
                      </div>
                    </form>
                  </Form>
                </div>
              )}

              {step === "finish" && (
                <div className="space-y-6">
                  <div className="grid gap-4 md:grid-cols-2">
                    <div className="rounded-lg border border-border p-4">
                      <p className="text-sm text-muted-foreground">Project</p>
                      <p className="mt-1 text-base font-medium">{project?.name}</p>
                    </div>
                    <div className="rounded-lg border border-border p-4">
                      <p className="text-sm text-muted-foreground">Team invites</p>
                      <p className="mt-1 text-base font-medium">
                        {!isEmailDeliveryAvailable
                          ? "Unavailable on this instance"
                          : inviteResults.length === 0
                            ? "No invites sent"
                            : `${createdInviteCount} sent${failedInviteCount > 0 ? `, ${failedInviteCount} need attention` : ""}`}
                      </p>
                    </div>
                  </div>

                  {inviteResults.length > 0 ? (
                    <div className="rounded-lg border border-border">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Email</TableHead>
                            <TableHead>Role</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Detail</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {inviteResults.map((result) => (
                            <TableRow key={`${result.email}-${result.role}`}>
                              <TableCell className="font-medium">{result.email}</TableCell>
                              <TableCell className="capitalize">{result.role}</TableCell>
                              <TableCell>
                                <Badge variant={getInviteResultVariant(result)}>
                                  {getInviteResultLabel(result)}
                                </Badge>
                              </TableCell>
                              <TableCell className="text-muted-foreground">
                                {result.error_message ?? "Invitation email queued successfully."}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  ) : (
                    <Alert>
                      <AlertTitle>
                        {isEmailDeliveryAvailable ? "No invites sent" : "Team invites unavailable"}
                      </AlertTitle>
                      <AlertDescription>
                        {isEmailDeliveryAvailable
                          ? "You can invite teammates later from your project settings."
                          : "Configure email delivery in Instance Settings to invite teammates later."}
                      </AlertDescription>
                    </Alert>
                  )}

                  <div className="flex justify-between gap-3">
                    <Button
                      variant="outline"
                      onClick={() => setStep(isEmailDeliveryAvailable ? "invite" : "project")}
                    >
                      Back
                    </Button>
                    <Button
                      onClick={() => void handleCompleteOnboarding()}
                      disabled={completeOnboarding.isPending}
                    >
                      {completeOnboarding.isPending ? "Finishing..." : "Finish setup"}
                    </Button>
                  </div>
                </div>
              )}
            </motion.div>
          </AnimatePresence>
        </CardContent>
      </Card>
    </div>
  );
}
