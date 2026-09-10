"use client";

import { useEffect, useRef, useState } from "react";
import { MoreHorizontal, Pencil, Plus, Trash2, MessagesSquare } from "lucide-react";
import type { Conversation } from "@/lib/chat-types";
import { providerClasses } from "./provider-meta";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

type Props = {
  conversations: Conversation[];
  selectedId: string | null;
  loading: boolean;
  onSelect: (id: string) => void;
  onCreate: () => void;
  onRename: (id: string, title: string) => void;
  onDelete: (id: string) => void;
};

function Row({
  conversation,
  selected,
  onSelect,
  onRename,
  onAskDelete,
}: {
  conversation: Conversation;
  selected: boolean;
  onSelect: () => void;
  onRename: (title: string) => void;
  onAskDelete: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(conversation.title);
  const inputRef = useRef<HTMLInputElement>(null);
  const c = providerClasses[conversation.provider];

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editing]);

  function commit() {
    const next = draft.trim();
    setEditing(false);
    if (next && next !== conversation.title) onRename(next);
    else setDraft(conversation.title);
  }

  if (editing) {
    return (
      <li className="px-2">
        <input
          ref={inputRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") commit();
            if (e.key === "Escape") {
              setDraft(conversation.title);
              setEditing(false);
            }
          }}
          aria-label="Conversation title"
          className="w-full rounded-md border bg-background px-2.5 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </li>
    );
  }

  return (
    <li className="group/row relative px-2">
      <button
        type="button"
        onClick={onSelect}
        onDoubleClick={() => setEditing(true)}
        aria-current={selected ? "true" : undefined}
        className={`flex w-full items-start gap-2.5 rounded-md py-2 pl-2.5 pr-9 text-left transition-colors ${
          selected ? "bg-accent" : "hover:bg-accent/50"
        }`}
      >
        <span
          className={`mt-1.5 size-1.5 shrink-0 rounded-full ${c.dot}`}
          aria-hidden
        />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm leading-snug">
            {conversation.title}
          </span>
          <span className="mt-0.5 block truncate text-xs text-muted-foreground">
            {conversation.folder.split("\\").pop()} · {conversation.updated}
          </span>
        </span>
      </button>

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon"
              aria-label={`Actions for ${conversation.title}`}
              className="absolute right-3 top-1.5 size-7 opacity-0 transition-opacity group-hover/row:opacity-100 data-[popup-open]:opacity-100 focus-visible:opacity-100"
            />
          }
        >
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-40">
          <DropdownMenuItem onClick={() => setEditing(true)}>
            <Pencil className="size-4" />
            Rename
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={onAskDelete}>
            <Trash2 className="size-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </li>
  );
}

export function ConversationList({
  conversations,
  selectedId,
  loading,
  onSelect,
  onCreate,
  onRename,
  onDelete,
}: Props) {
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null);

  return (
    <aside className="flex h-full w-72 shrink-0 flex-col border-r bg-card">
      <div className="flex shrink-0 items-center justify-between gap-2 px-4 py-3">
        <h2 className="text-sm font-medium">Conversations</h2>
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          onClick={onCreate}
          aria-label="New conversation"
        >
          <Plus className="size-4" />
        </Button>
      </div>

      {loading ? (
        <ul className="space-y-1 px-2" aria-busy="true" aria-label="Loading conversations">
          {[68, 52, 60, 44].map((w, i) => (
            <li key={i} className="space-y-1.5 px-2.5 py-2">
              <Skeleton className="h-3.5" style={{ width: `${w}%` }} />
              <Skeleton className="h-3 w-1/3" />
            </li>
          ))}
        </ul>
      ) : conversations.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
          <MessagesSquare
            className="size-7 text-muted-foreground"
            aria-hidden
          />
          <p className="text-sm text-muted-foreground">
            No conversations yet. Start one and it will be saved here — the
            transcript is kept by sparstrowgen, not by whichever agent answered.
          </p>
          <Button size="sm" onClick={onCreate}>
            <Plus className="size-4" />
            New conversation
          </Button>
        </div>
      ) : (
        <ScrollArea className="min-h-0 flex-1">
          <ul className="space-y-0.5 pb-3">
            {conversations.map((c) => (
              <Row
                key={c.id}
                conversation={c}
                selected={c.id === selectedId}
                onSelect={() => onSelect(c.id)}
                onRename={(title) => onRename(c.id, title)}
                onAskDelete={() => setPendingDelete(c)}
              />
            ))}
          </ul>
        </ScrollArea>
      )}

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => !open && setPendingDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this conversation?</AlertDialogTitle>
            <AlertDialogDescription>
              &ldquo;{pendingDelete?.title}&rdquo; and its full transcript will
              be removed. Since the transcript lives here rather than with any
              agent, this cannot be recovered from the provider.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (pendingDelete) onDelete(pendingDelete.id);
                setPendingDelete(null);
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </aside>
  );
}
