import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Label } from "@/components/ui/label.tsx";
import { useProjectDetail, useDeleteProject } from "@/hooks/use-projects.ts";
import { useUIStore } from "@/stores/ui-store.ts";

// ─── Delete confirmation dialog ───────────────────────────────────────────────

function DeleteProjectDialog({
  open,
  onOpenChange,
  projectName,
  onConfirm,
  loading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectName: string;
  onConfirm: () => void;
  loading: boolean;
}) {
  const [typed, setTyped] = useState("");

  function handleOpenChange(next: boolean) {
    onOpenChange(next);
    if (!next) setTyped("");
  }

  const confirmed = typed === projectName;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete project</DialogTitle>
          <DialogDescription>
            This action is permanent and cannot be undone. All nodes, services,
            alert rules, and history will be permanently removed.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 py-1">
          <Label className="text-sm text-muted-foreground">
            Type{" "}
            <span className="font-mono font-semibold text-foreground">
              {projectName}
            </span>{" "}
            to confirm
          </Label>
          <Input
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            placeholder={projectName}
            autoComplete="off"
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={!confirmed || loading}
          >
            {loading ? "Deleting…" : "Delete project"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── Section ──────────────────────────────────────────────────────────────────

export function DangerZoneSection({ projectId }: { projectId: number }) {
  const navigate = useNavigate();
  const [deleteOpen, setDeleteOpen] = useState(false);

  const { data } = useProjectDetail(projectId);
  const deleteProject = useDeleteProject();
  const setActiveProjectId = useUIStore((s) => s.setActiveProjectId);

  const projectName = data?.project.name ?? "";

  async function handleDelete() {
    try {
      await deleteProject.mutateAsync(projectId);
      setActiveProjectId(null);
      toast.success("Project deleted");
      navigate("/overview");
    } catch {
      toast.error("Failed to delete project. Please try again.");
    }
  }

  return (
    <div className="space-y-4">

      {/* Section label */}
      <div>
        <h2 className="text-sm font-semibold">Danger zone</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          These actions are permanent and cannot be undone.
        </p>
      </div>

      {/* Delete project row */}
      <div className="flex items-center justify-between gap-6 rounded-xl border border-destructive/20 bg-destructive/5 px-5 py-4">
        <div className="space-y-0.5 min-w-0">
          <p className="text-sm font-medium text-foreground">Delete this project</p>
          <p className="text-sm text-muted-foreground">
            Permanently removes all nodes, services, alert rules, and history. This cannot be reversed.
          </p>
        </div>
        <Button
          variant="destructive"
          size="sm"
          onClick={() => setDeleteOpen(true)}
          className="shrink-0"
        >
          Delete project
        </Button>
      </div>

      <DeleteProjectDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        projectName={projectName}
        onConfirm={handleDelete}
        loading={deleteProject.isPending}
      />
    </div>
  );
}
