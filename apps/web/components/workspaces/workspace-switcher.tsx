"use client";

import Link from "next/link";
import { Check, ChevronsUpDown, FolderOpen, Settings2 } from "lucide-react";
import { cn } from "cn";
import { useCurrentWorkspace, useWorkspaces } from "@/lib/queries";
import { useWorkspaceView } from "@/lib/store";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

/* Which area of work you are in, and the way to be in another one
   (docs/Decisions.md D-050).

   It sits at the top of the rail, above the sections, because it scopes them
   rather than being one of them: everything below it is "in" whichever
   workspace this names. The same reason it is not a fourth nav entry — Chat is
   a place you go, a workspace is the world you are in when you get there.

   With one workspace it is still shown. Hiding it until there are two would
   mean the first switch happens through a control nobody has ever seen, and it
   is also the only place that says which workspace the conversations on screen
   belong to. */

export function WorkspaceSwitcher({ pinned }: { pinned: boolean }) {
  const workspaces = useWorkspaces();
  const currentId = useCurrentWorkspace();
  const setWorkspace = useWorkspaceView((s) => s.setWorkspace);

  const list = workspaces.data ?? [];
  const current = list.find((w) => w.id === currentId);

  // Nothing to switch between until the account has one. This is a brand-new
  // account on its way to the setup wizard, not a state to draw a control for.
  if (!current) return null;

  return (
    <div className="px-2 pt-2">
      <DropdownMenu>
        <DropdownMenuTrigger
          className={cn(
            "flex w-full items-center gap-3 rounded-md px-2.5 py-2 text-sm whitespace-nowrap",
            "hover:bg-accent/60 focus-visible:outline-2 focus-visible:outline-ring",
          )}
          aria-label={`Workspace: ${current.name}. Switch workspace`}
        >
          <FolderOpen className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          <span
            className={cn(
              "flex min-w-0 flex-1 items-center justify-between gap-2",
              "transition-opacity duration-150 motion-reduce:transition-none",
              pinned
                ? "opacity-100"
                : "opacity-0 group-hover/rail:opacity-100 group-focus-within/rail:opacity-100",
            )}
          >
            <span className="min-w-0 truncate font-medium" title={current.name}>
              {current.name}
            </span>
            <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </span>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="start" className="w-56">
          {list.map((w) => (
            <DropdownMenuItem
              key={w.id}
              onClick={() => setWorkspace(w.id)}
              className="justify-between gap-3"
            >
              <span className="min-w-0 truncate">{w.name}</span>
              {w.id === current.id && <Check className="size-4 shrink-0" aria-hidden />}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          {/* Making and renaming happen in Settings rather than in this menu.
              A switcher that also creates is a menu you cannot open without
              risking the thing you did not mean to do. */}
          <DropdownMenuItem render={<Link href="/settings/workspaces" />}>
            <Settings2 />
            Manage workspaces
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
