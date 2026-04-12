import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { MoreHorizontalIcon, PlusIcon, UserIcon } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar.tsx";
import { Badge } from "@/components/ui/badge.tsx";
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
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
import { Separator } from "@/components/ui/separator.tsx";
import { useAuth } from "@/hooks/use-auth.ts";
import {
  useInviteMembers,
  useProjectInvitations,
  useProjectMembers,
  useRemoveMember,
  useRevokeInvitation,
  useUpdateMemberRole,
} from "@/hooks/use-project-members.ts";
import { useCanManageMembers } from "@/hooks/use-project-role.ts";
import type { ProjectMember, ProjectMemberRole } from "@/types/project-member.ts";
import type { ProjectInvitationRole } from "@/types/project-invitation.ts";

// ─── Role labels / badge styles ──────────────────────────────────────────────

const roleLabel: Record<string, string> = {
  owner: "Owner",
  admin: "Admin",
  viewer: "Viewer",
};

const roleBadgeVariant: Record<string, "default" | "secondary" | "outline"> = {
  owner: "default",
  admin: "secondary",
  viewer: "outline",
};

// ─── Invite dialog ────────────────────────────────────────────────────────────

const inviteSchema = z.object({
  email: z.string().email("Enter a valid email"),
  role: z.enum(["admin", "viewer"]),
});
type InviteFormValues = z.infer<typeof inviteSchema>;

function InviteDialog({
  projectId,
  open,
  onOpenChange,
}: {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const invite = useInviteMembers(projectId);

  const form = useForm<InviteFormValues>({
    resolver: zodResolver(inviteSchema),
    defaultValues: { email: "", role: "viewer" },
  });

  async function onSubmit(values: InviteFormValues) {
    try {
      const result = await invite.mutateAsync({
        project_id: projectId,
        invitations: [{ email: values.email, role: values.role }],
      });

      const item = result.results[0];
      if (item?.status === "ok") {
        toast.success(`Invitation sent to ${values.email}`);
        form.reset();
        onOpenChange(false);
      } else if (item?.error_code === "already_member") {
        toast.error("This user is already a member of the project.");
      } else if (item?.error_code === "already_invited") {
        toast.error("This email has already been invited.");
      } else {
        toast.error(item?.error_message ?? "Failed to send invitation.");
      }
    } catch {
      toast.error("Failed to send invitation. Please try again.");
    }
  }

  function handleOpenChange(next: boolean) {
    onOpenChange(next);
    if (!next) form.reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Invite member</DialogTitle>
          <DialogDescription>
            Send an invitation to a user to join this project.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div className="flex gap-3">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input
                        type="email"
                        placeholder="name@example.com"
                        autoComplete="off"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem className="w-28 shrink-0">
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="admin">Admin</SelectItem>
                        <SelectItem value="viewer">Viewer</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <DialogFooter>
              <Button variant="outline" type="button" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={invite.isPending}>
                {invite.isPending ? "Sending…" : "Send invite"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

// ─── Member row ───────────────────────────────────────────────────────────────

function MemberRow({
  member,
  isCurrentUser,
  canManage,
  projectId,
}: {
  member: ProjectMember;
  isCurrentUser: boolean;
  canManage: boolean;
  projectId: number;
}) {
  const updateRole = useUpdateMemberRole(projectId);
  const removeMember = useRemoveMember(projectId);

  const initials = member.user.name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  async function handleRoleChange(role: string) {
    try {
      await updateRole.mutateAsync({ id: member.id, role: role as ProjectMemberRole });
      toast.success("Role updated");
    } catch {
      toast.error("Failed to update role.");
    }
  }

  async function handleRemove() {
    try {
      await removeMember.mutateAsync(member.id);
      toast.success("Member removed");
    } catch (err: any) {
      const msg = err?.message ?? "";
      if (msg.includes("last") || msg.includes("owner")) {
        toast.error("Cannot remove the last owner.");
      } else {
        toast.error("Failed to remove member.");
      }
    }
  }

  return (
    <div className="flex items-center gap-3 px-6 py-3.5">
      <Avatar className="size-8 shrink-0">
        {member.user.image && <AvatarImage src={member.user.image} alt={member.user.name} />}
        <AvatarFallback className="text-xs">{initials}</AvatarFallback>
      </Avatar>

      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">
          {member.user.name}
          {isCurrentUser && (
            <span className="ml-1.5 text-xs text-muted-foreground font-normal">(you)</span>
          )}
        </p>
        <p className="text-xs text-muted-foreground truncate">{member.user.email}</p>
      </div>

      <Badge variant={roleBadgeVariant[member.role] ?? "outline"} className="shrink-0 text-xs">
        {roleLabel[member.role] ?? member.role}
      </Badge>

      {canManage && member.role !== "owner" && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon-sm" className="shrink-0 hover:bg-secondary hover:text-secondary-foreground text-muted-foreground/60">
              <MoreHorizontalIcon size={15} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              disabled={member.role === "admin"}
              onSelect={() => handleRoleChange("admin")}
            >
              Make admin
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={member.role === "viewer"}
              onSelect={() => handleRoleChange("viewer")}
            >
              Make viewer
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onSelect={handleRemove}
            >
              Remove member
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}

// ─── Pending invitations ──────────────────────────────────────────────────────

function PendingInvitationsCard({
  projectId,
  canManage,
}: {
  projectId: number;
  canManage: boolean;
}) {
  const { data } = useProjectInvitations(projectId);
  const revokeInvitation = useRevokeInvitation(projectId);

  const pending = (data?.project_invitations ?? []).filter((inv) => !inv.accepted_at);

  if (pending.length === 0) return null;

  async function handleRevoke(id: string) {
    try {
      await revokeInvitation.mutateAsync(id);
      toast.success("Invitation revoked");
    } catch {
      toast.error("Failed to revoke invitation.");
    }
  }

  return (
    <div className="rounded-xl border border-border overflow-hidden">
      <div className="px-6 py-5">
        <h2 className="text-sm font-semibold">Pending invitations</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          Awaiting acceptance.
        </p>
      </div>
      <Separator />
      <div className="divide-y divide-border">
        {pending.map((inv) => (
          <div key={inv.id} className="flex items-center gap-3 px-6 py-3.5">
            <div className="flex size-8 items-center justify-center rounded-full bg-muted shrink-0">
              <UserIcon size={14} className="text-muted-foreground" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{inv.email}</p>
              <p className="text-xs text-muted-foreground">
                Expires {new Date(inv.expires_at).toLocaleDateString()}
              </p>
            </div>
            <Badge variant={roleBadgeVariant[inv.role as ProjectInvitationRole] ?? "outline"} className="shrink-0 text-xs">
              {roleLabel[inv.role] ?? inv.role}
            </Badge>
            {canManage && (
              <Button
                variant="ghost"
                size="sm"
                className="shrink-0 text-muted-foreground/60 hover:text-destructive"
                onClick={() => handleRevoke(inv.id)}
                disabled={revokeInvitation.isPending}
              >
                Revoke
              </Button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Section ──────────────────────────────────────────────────────────────────

export function MembersSection({ projectId }: { projectId: number }) {
  const [inviteOpen, setInviteOpen] = useState(false);
  const { user } = useAuth();
  const canManage = useCanManageMembers(projectId);
  const { data, isLoading } = useProjectMembers(projectId);

  const members = data?.project_members ?? [];

  return (
    <div className="space-y-6">
      {/* Members card */}
      <div className="rounded-xl border border-border overflow-hidden">
        <div className="flex items-center justify-between px-6 py-5">
          <div>
            <h2 className="text-sm font-semibold">Members</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              People with access to this project.
            </p>
          </div>
          {canManage && (
            <Button size="sm" onClick={() => setInviteOpen(true)}>
              <PlusIcon size={14} />
              Invite
            </Button>
          )}
        </div>

        <Separator />

        {isLoading ? (
          <div className="px-6 py-10 text-center text-sm text-muted-foreground">Loading…</div>
        ) : members.length === 0 ? (
          <div className="px-6 py-10 text-center text-sm text-muted-foreground">
            No members found.
          </div>
        ) : (
          <div className="divide-y divide-border">
            {members.map((member) => (
              <MemberRow
                key={member.id}
                member={member}
                isCurrentUser={member.user_id === user?.id}
                canManage={canManage}
                projectId={projectId}
              />
            ))}
          </div>
        )}
      </div>

      {/* Pending invitations */}
      <PendingInvitationsCard projectId={projectId} canManage={canManage} />

      {/* Invite dialog */}
      <InviteDialog
        projectId={projectId}
        open={inviteOpen}
        onOpenChange={setInviteOpen}
      />
    </div>
  );
}
