import { LogOutIcon, MoonIcon, SunIcon, UserIcon } from "@/components/animate-ui/icons";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { useAuth } from "@/hooks/use-auth.ts";
import { useTheme } from "@/hooks/use-theme.ts";
import { getInitials } from "@/lib/utils.ts";
import { AnimateIcon } from "../animate-ui/icons/icon";
import { EllipsisIcon } from "lucide-react";
import { Link } from "react-router-dom";

export function Topbar() {
  const { user, logout } = useAuth();
  const { resolvedTheme, setTheme } = useTheme();
  // const toggleSidebar = useUIStore((s) => s.toggleSidebar);

  function toggleTheme() {
    setTheme(resolvedTheme === "dark" ? "light" : "dark");
  }

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-4 shrink-0">
      <span />

      <div className="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="hover:bg-secondary flex items-center gap-0.5 hover:text-foreground focus:text-foreground"
            >
              <Avatar className="size-7">
                <AvatarImage src={user?.image ?? undefined} />
                <AvatarFallback className="text-[11px] font-semibold bg-accent text-accent-foreground">
                  {user ? getInitials(user.name) : <UserIcon size={14} />}
                </AvatarFallback>
              </Avatar>
              <div className="ml-2 hidden items-center gap-4 sm:flex">
                <span className="text-sm font-medium">{user?.name}</span>
                <EllipsisIcon size={16} className="opacity-50" />
              </div>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="min-w-48">
            <DropdownMenuItem asChild>
              <AnimateIcon asChild animateOnHover>
                <Link to="/profile" className="flex items-center gap-2">
                  <UserIcon size={14} className="mr-2" />
                  Profile
                </Link>
              </AnimateIcon>
            </DropdownMenuItem>
            <DropdownMenuItem
              asChild
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                toggleTheme();
              }}
            >
              <AnimateIcon asChild animateOnHover>
                <span className="flex items-center gap-2">
                  {resolvedTheme === "dark" ? (
                    <SunIcon size={16} className="mr-2" />
                  ) : (
                    <MoonIcon size={16} className="mr-2" />
                  )}
                  Toggle Theme
                </span>
              </AnimateIcon>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => void logout()}
              className="text-destructive hover:bg-destructive/50 hover:text-destructive-foreground focus:bg-destructive/50! focus:text-foreground!"
              asChild
            >
              <AnimateIcon asChild animateOnHover>
                <span className="flex items-center gap-2">
                  <LogOutIcon size={14} className="mr-2" />
                  Sign out
                </span>
              </AnimateIcon>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
