import { useEffect, useState } from "react";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectLabel,
  SelectSeparator,
} from "@/components/ui/select.tsx";
import { useProjects, useCreateProject } from "@/hooks/use-projects.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { motion } from "motion/react";
import { SelectTrigger, SelectItem } from "@radix-ui/react-select";
import { cn } from "@/lib/utils";
import { ChevronUpDownIcon, PlusIcon, AnimateIcon } from "../animate-ui/icons";
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
import { Button } from "@/components/ui/button.tsx";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { ApiError, NetworkError } from "@/lib/fetch-client.ts";

const createProjectSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
});

type CreateProjectFormValues = z.infer<typeof createProjectSchema>;

function getCreateProjectErrorMessage(error: unknown) {
  if (error instanceof NetworkError) {
    return error.message;
  }
  if (error instanceof ApiError) {
    if (error.status === 409) {
      return "A project with this name already exists.";
    }
    if (error.status === 400) {
      return "Review the project details and try again.";
    }
    if (error.status >= 500) {
      return "Agrafa couldn't create the project right now. Try again in a moment.";
    }
  }
  return "Couldn't create the project. Try again.";
}

function CreateProjectDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (id: number) => void;
}) {
  const createProject = useCreateProject();
  const form = useForm<CreateProjectFormValues>({
    resolver: zodResolver(createProjectSchema),
    defaultValues: { name: "" },
  });

  useEffect(() => {
    if (!open) {
      form.reset({ name: "" });
    }
  }, [open, form]);

  async function onSubmit(values: CreateProjectFormValues) {
    try {
      const result = await createProject.mutateAsync({ name: values.name.trim() });
      toast.success("Project created");
      onOpenChange(false);
      onCreated(result.project.id);
    } catch (error) {
      toast.error(getCreateProjectErrorMessage(error));
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>New project</DialogTitle>
          <DialogDescription>Give your project a name to get started.</DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Project name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g. Production" autoFocus {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" type="button" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createProject.isPending}>
                {createProject.isPending ? "Creating..." : "Create"}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

export function ProjectSwitcher({ isSidebarOpen = true }: { isSidebarOpen?: boolean }) {
  const { data } = useProjects();
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const setActiveProjectId = useUIStore((s) => s.setActiveProjectId);
  const [createOpen, setCreateOpen] = useState(false);

  // Auto-select first project if none selected
  useEffect(() => {
    if (!activeProjectId && data?.projects.length) {
      setActiveProjectId(data.projects[0]!.id);
    }
  }, [data, activeProjectId, setActiveProjectId]);

  const projects = data?.projects ?? [];

  function handleValueChange(val: string) {
    if (val === "create_new") {
      setCreateOpen(true);
      return;
    }
    setActiveProjectId(Number(val));
  }

  return (
    <>
      <Select value={activeProjectId?.toString() ?? ""} onValueChange={handleValueChange}>
        <SelectTrigger asChild>
          <AnimateIcon asChild animateOnHover>
            <motion.button
              animate={{ opacity: projects.length ? 1 : 0.5 }}
              disabled={!projects.length}
              className={cn(
                "flex h-10 w-full items-center rounded-md hover:bg-sidebar-primary/50 data-placeholder:text-muted-foreground",
                isSidebarOpen ? "px-2" : "px-1",
              )}
            >
              <span className="size-7 mr-2 rounded-md bg-accent text-xs font-semibold flex items-center justify-center shrink-0">
                {projects.find((p) => p.id === activeProjectId)?.name[0] ?? "P"}
              </span>
              {isSidebarOpen && (
                <>
                  <span className="text-sm font-medium">
                    {projects.find((p) => p.id === activeProjectId)?.name || "Select Project"}
                  </span>
                  <ChevronUpDownIcon size={16} className="ml-auto opacity-50" />
                </>
              )}
            </motion.button>
          </AnimateIcon>
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectLabel className="text-xs text-muted-foreground leading-2">Projects</SelectLabel>
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

      <CreateProjectDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(id) => setActiveProjectId(id)}
      />
    </>
  );
}
