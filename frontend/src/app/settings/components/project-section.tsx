import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { Button } from "@/components/ui/button.tsx";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Separator } from "@/components/ui/separator.tsx";
import { useProjectDetail, useUpdateProject } from "@/hooks/use-projects.ts";
import { useCanWrite } from "@/hooks/use-project-role.ts";

const schema = z.object({ name: z.string().min(1, "Name is required") });
type FormValues = z.infer<typeof schema>;

export function ProjectSection({ projectId }: { projectId: number }) {
  const { data } = useProjectDetail(projectId);
  const update = useUpdateProject();
  const canWrite = useCanWrite(projectId);

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  });

  useEffect(() => {
    if (data?.project.name) form.reset({ name: data.project.name });
  }, [data, form]);

  async function onSubmit(values: FormValues) {
    await update.mutateAsync({ id: projectId, payload: { name: values.name } });
    toast.success("Project updated");
  }

  return (
    <div className="space-y-6">
      <div className="rounded-xl border border-border overflow-hidden">
        <div className="px-6 py-5">
          <h2 className="text-sm font-semibold">Project name</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Update the display name for this project.
          </p>
        </div>
        <Separator />
        <div className="px-6 py-5">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 max-w-md">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input {...field} disabled={!canWrite} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {canWrite && (
                <Button type="submit" size="sm" disabled={update.isPending}>
                  {update.isPending ? "Saving…" : "Save"}
                </Button>
              )}
            </form>
          </Form>
        </div>
      </div>
    </div>
  );
}
