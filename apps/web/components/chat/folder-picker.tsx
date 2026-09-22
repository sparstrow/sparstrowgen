"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ChevronRight,
  Clock,
  CornerLeftUp,
  Folder,
  GitBranch,
  HardDrive,
  Loader2,
} from "lucide-react";
import type { DirListing, DirReason } from "@/lib/chat-types";
import { api } from "@/lib/api";
import { useCurrentWorkspace } from "@/lib/queries";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Status, StatusIcon } from "@/components/ui/status";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";

/* Which directory a conversation runs in.
 *
 * The folder is what the agent can see, so a conversation pointed at the wrong
 * one gives confidently wrong answers about the wrong codebase with nothing on
 * screen saying so (docs/Bugs.md B-3). That is why this is a real picker rather
 * than a text field: a path typed by hand is how that failure recurs.
 *
 * A browser cannot open a native folder dialog and cannot see the machine, so
 * every path question is answered by the daemon (docs/Decisions.md D-020).
 */

/** The wording for a failed check lives here, in the surface that shows it,
 *  rather than on the daemon — so it can be rewritten without a protocol
 *  change, and a client never parses prose to tell one failure from another. */
const REASONS: Record<Exclude<DirReason, "">, string> = {
  not_absolute: "That is not a full path.",
  not_found: "There is nothing at that path.",
  not_a_directory: "That is a file. Pick the folder it is in.",
  not_readable: "That folder cannot be opened.",
};

function Row({
  icon,
  label,
  hint,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  hint?: React.ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-sm hover:bg-accent"
    >
      {icon}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {hint}
    </button>
  );
}

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Where the conversation runs now. The picker opens here rather than at a
   *  root, because the reason for opening it is usually to move somewhere
   *  nearby. */
  current: string;
  onChoose: (folder: string) => void;
  /** Changing folder once answers exist means the transcript above came from
   *  somewhere else. Allowed — that is usually when the mistake is noticed —
   *  but said out loud. */
  hasMessages: boolean;
};

/** The dialog shell. Browsing state lives in Body, which is mounted fresh on
 *  each opening and keyed by the current folder — so reopening starts where the
 *  conversation actually is, without an effect writing state on every open. */
export function FolderPicker(props: Props) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="flex max-h-[80vh] flex-col gap-4 sm:max-w-2xl">
        {props.open && <Body key={props.current} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

function Body({ onOpenChange, current, onChoose, hasMessages }: Props) {
  const [path, setPath] = useState(current);
  const [typed, setTyped] = useState(current);

  // Both queries are per workspace. The folders somebody works in say as much
  // about what they are doing as the conversations do, so a personal workspace
  // must not suggest a client's directory (D-050) — and browsing goes to the
  // computer THIS workspace would run on, so a path picked here is one the next
  // message can actually reach (00018).
  const workspaceId = useCurrentWorkspace();

  const listing = useQuery<DirListing>({
    queryKey: ["directories", workspaceId, path],
    queryFn: () => api.directories(workspaceId!, path),
    enabled: workspaceId !== null,
    // Directories change under us, and nothing here is worth reusing between
    // openings.
    staleTime: 0,
    retry: false,
  });

  const recents = useQuery({
    queryKey: ["recent-folders", workspaceId],
    queryFn: () => api.recentFolders(workspaceId!),
    enabled: workspaceId !== null,
  });

  const data = listing.data;
  const usable = data?.reason === "";
  const shownRecents = (recents.data ?? [])
    .filter((f) => f !== data?.path)
    .slice(0, 5);

  function go(next: string) {
    setPath(next);
    setTyped(next);
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>Working directory</DialogTitle>
        <DialogDescription>
          The folder the agent runs in, and the only code it can see.
        </DialogDescription>
      </DialogHeader>

      {/* A visible action, not just Enter. A pasted path with no button beside
          it leaves the only way to apply it invisible. */}
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          go(typed.trim());
        }}
      >
        <Input
          value={typed}
          onChange={(e) => setTyped(e.target.value)}
          spellCheck={false}
          aria-label="Path"
          placeholder="Paste a path, or browse below"
          className="font-mono text-xs"
        />
        <Button
          type="submit"
          variant="outline"
          disabled={!typed.trim() || typed.trim() === data?.path}
        >
          Go
        </Button>
      </form>

      {shownRecents.length > 0 && (
        <div>
          <p className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="size-3.5" aria-hidden />
            Recent
          </p>
          <div className="flex flex-wrap gap-1.5">
            {shownRecents.map((f) => (
              <Button
                key={f}
                variant="outline"
                size="sm"
                className="h-7 max-w-full font-mono text-xs"
                onClick={() => go(f)}
              >
                <span className="truncate">{f}</span>
              </Button>
            ))}
          </div>
        </div>
      )}

      <ScrollArea className="min-h-40 flex-1 rounded-lg border">
        <div className="p-1.5">
          {listing.isPending ? (
            <p className="flex items-center gap-2 px-2.5 py-6 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" aria-hidden />
              Reading your machine…
            </p>
          ) : listing.isError ? (
            <div className="px-2.5 py-6 text-sm">
              <p className="font-medium">Could not read that</p>
              <p className="mt-0.5 text-muted-foreground">
                {(listing.error as Error).message}
              </p>
            </div>
          ) : (
            <>
              {data?.parent ? (
                <Row
                  icon={
                    <CornerLeftUp className="size-4 shrink-0 text-muted-foreground" />
                  }
                  label="Up one level"
                  onClick={() => go(data.parent)}
                />
              ) : (
                <p className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs text-muted-foreground">
                  <HardDrive className="size-3.5" aria-hidden />
                  Where to start on this machine
                </p>
              )}

              {!usable && data?.reason && (
                <p className="px-2.5 py-3 text-sm text-muted-foreground">
                  <Status tone="danger" quiet size="md">
                    {REASONS[data.reason]}
                  </Status>
                </p>
              )}

              {usable && data.entries.length === 0 && (
                <p className="px-2.5 py-6 text-sm text-muted-foreground">
                  No folders in here. It can still be chosen.
                </p>
              )}

              {data?.entries.map((e) => (
                <Row
                  key={e.path}
                  icon={
                    <Folder className="size-4 shrink-0 text-muted-foreground" />
                  }
                  label={e.name}
                  hint={
                    <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
                  }
                  onClick={() => go(e.path)}
                />
              ))}
            </>
          )}
        </div>
      </ScrollArea>

      {usable && (
        <p className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
          {data.isGitRepo ? (
            <GitBranch className="size-3.5 shrink-0" aria-hidden />
          ) : (
            <StatusIcon tone="warning" />
          )}
          <span className="truncate">
            {data.path}
            {!data.isGitRepo && " — not inside a git repository"}
          </span>
        </p>
      )}

      {hasMessages && (
        <p className="text-xs text-muted-foreground">
          This conversation already has answers in it. They were about the old
          folder, and nothing above will say so. Moving it starts each agent
          again from scratch, so the next message replays the conversation.
        </p>
      )}

      <DialogFooter>
        <Button variant="ghost" onClick={() => onOpenChange(false)}>
          Cancel
        </Button>
        <Button
          disabled={!usable || data.path === current}
          onClick={() => {
            if (data) onChoose(data.path);
            onOpenChange(false);
          }}
        >
          Run here
        </Button>
      </DialogFooter>
    </>
  );
}
