"use client";

import { useState } from "react";
import { Check, FolderOpen, Plus } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import {
  useCreateWorkspace,
  useCurrentWorkspace,
  useRenameWorkspace,
  useWorkspaces,
} from "@/lib/queries";
import { useWorkspaceView } from "@/lib/store";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

/* The workspace editor, shared by Settings → Workspaces and first-run setup's
   second step — one form rather than two that drift, the same reasoning as
   ProfileFields and D-046.

   A workspace has one editable thing: its name. What it holds is edited through
   the things it holds. */

export const WORKSPACE_NAME_LIMIT = 60;

/** The suggestion in an empty account's box. The owner's own split was "one
 *  personal and one work related workspace", so the first one is Personal and
 *  the prompt to add another says Work. */
const FIRST_NAME = "Personal";

export function WorkspaceFields({ onCreated }: { onCreated?: () => void }) {
  const workspaces = useWorkspaces();
  const current = useCurrentWorkspace();
  const create = useCreateWorkspace();

  const list = workspaces.data;
  const none = (list ?? []).length === 0;

  // The "add another" box is closed until asked for, once there is something to
  // add to. With nothing yet it IS the screen, so there is nothing to open.
  const [adding, setAdding] = useState(false);
  // Pre-filled with the suggestion only for the FIRST one, where "Personal" is
  // a sensible default somebody can accept with one press. A second workspace
  // has no obvious name, and starting it at "Personal" meant typing straight
  // onto the end of it.
  const [name, setName] = useState(FIRST_NAME);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const wanted = name.trim();
    if (!wanted) return;
    create.mutate(wanted, {
      onSuccess: (made) => {
        toast.success(`${made.name} is ready`);
        setName("");
        setAdding(false);
        onCreated?.();
      },
    });
  }

  if (workspaces.isPending) {
    return (
      <div className="space-y-3" aria-busy="true">
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-2/3" />
      </div>
    );
  }

  if (workspaces.isError) {
    return (
      <p className="text-sm text-muted-foreground">
        Your workspaces could not be loaded: {(workspaces.error as Error).message}
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {!none && (
        <ul className="divide-y rounded-lg border">
          {list!.map((w) => (
            <li key={w.id}>
              <WorkspaceRow
                id={w.id}
                name={w.name}
                canRename={w.role === "owner"}
                isCurrent={w.id === current}
              />
            </li>
          ))}
        </ul>
      )}

      {none || adding ? (
        <form onSubmit={submit} className="space-y-2">
          {none && (
            <label htmlFor="workspace-name" className="block text-sm font-medium">
              Name this workspace
            </label>
          )}
          <div className="flex flex-wrap gap-2">
            <Input
              id="workspace-name"
              value={name}
              autoFocus={adding}
              maxLength={WORKSPACE_NAME_LIMIT}
              onChange={(e) => setName(e.target.value)}
              placeholder={none ? FIRST_NAME : "Work"}
              aria-label={none ? undefined : "Name of the new workspace"}
              className="min-w-48 flex-1"
            />
            <Button type="submit" disabled={!name.trim() || create.isPending}>
              {create.isPending ? "Creating…" : none ? "Create workspace" : "Add"}
            </Button>
            {!none && (
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setAdding(false);
                  setName("");
                }}
              >
                Cancel
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            {none
              ? "Everything you do lives in a workspace. Most people start with one and add another when a second body of work turns up."
              : "A separate area of work. Its conversations and folders stay out of the others."}
          </p>
        </form>
      ) : (
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setName("");
            setAdding(true);
          }}
        >
          <Plus />
          Add a workspace
        </Button>
      )}
    </div>
  );
}

/** One row: its name, editable in place by whoever owns it.
 *
 *  Saved on blur as well as on Enter, because a name typed and then clicked
 *  away from is a name somebody meant. Escape puts back what was there. */
function WorkspaceRow({
  id,
  name,
  canRename,
  isCurrent,
}: {
  id: string;
  name: string;
  canRename: boolean;
  isCurrent: boolean;
}) {
  const rename = useRenameWorkspace();
  const setWorkspace = useWorkspaceView((s) => s.setWorkspace);
  const [draft, setDraft] = useState(name);

  // The saved name is the source of truth; a rename from another tab arrives
  // over the socket and has to win over a box nobody is typing in. Adjusting
  // during render rather than in an effect, as ProfileFields does.
  const [seen, setSeen] = useState(name);
  if (name !== seen) {
    setSeen(name);
    setDraft(name);
  }

  function save() {
    const wanted = draft.trim();
    if (!wanted || wanted === name) {
      setDraft(name);
      return;
    }
    rename.mutate({ id, name: wanted });
  }

  return (
    <div className="flex items-center gap-3 px-3 py-2.5">
      <FolderOpen className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      {canRename ? (
        <Input
          value={draft}
          maxLength={WORKSPACE_NAME_LIMIT}
          aria-label={`Name of ${name}`}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={save}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              e.currentTarget.blur();
            }
            if (e.key === "Escape") {
              setDraft(name);
              e.currentTarget.blur();
            }
          }}
          className={cn(
            "h-8 border-transparent bg-transparent px-2 shadow-none",
            "hover:border-input focus-visible:border-input focus-visible:bg-background",
          )}
        />
      ) : (
        <span className="min-w-0 flex-1 truncate px-2 text-sm">{name}</span>
      )}
      {isCurrent ? (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
          <Check className="size-3.5" aria-hidden />
          Open
        </span>
      ) : (
        <Button
          variant="ghost"
          size="sm"
          className="shrink-0"
          onClick={() => setWorkspace(id)}
        >
          Open
        </Button>
      )}
    </div>
  );
}
