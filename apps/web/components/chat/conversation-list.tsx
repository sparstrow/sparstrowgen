"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import {
  Archive,
  ArchiveRestore,
  ChevronRight,
  MessagesSquare,
  MoreHorizontal,
  Pencil,
  Plus,
  Search,
  Trash2,
  X,
} from "lucide-react";
import type { Conversation } from "@/lib/chat-types";
import { useChatView } from "@/lib/store";
import { providerStyle } from "./provider-meta";
import { ProviderIcon } from "./provider-icon";
import { SignOutMenu } from "@/components/auth/sign-out";
import {
  ConversationName,
  conversationName,
  unnamedConversation,
} from "./conversation-name";
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
  onSetArchived: (id: string, archived: boolean) => void;
};

function Row({
  conversation,
  showExcerpt,
  selected,
  onSelect,
  onRename,
  onSetArchived,
  onAskDelete,
}: {
  conversation: Conversation;
  showExcerpt: boolean;
  selected: boolean;
  onSelect: () => void;
  onRename: (title: string) => void;
  onSetArchived: (archived: boolean) => void;
  onAskDelete: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(conversation.title);
  const inputRef = useRef<HTMLInputElement>(null);
  const c = providerStyle(conversation.provider);

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
          placeholder={unnamedConversation}
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
        <ProviderIcon
          provider={conversation.provider}
          className={`mt-0.5 size-3.5 shrink-0 ${c.text} ${
            conversation.archived ? "opacity-60" : ""
          }`}
        />
        <span className="min-w-0 flex-1">
          <ConversationName
            title={conversation.title}
            className="block truncate text-sm leading-snug"
          />
          <span className="mt-0.5 block truncate text-xs text-muted-foreground">
            {conversation.folder.split("\\").pop()} · {conversation.updated}
          </span>
          {/* Only ever shown for a match found in the message text — a title
              match explains itself, and repeating it would say nothing. The
              server decides which, and sends an excerpt only for the former. */}
          {showExcerpt && conversation.excerpt && (
            <span className="mt-1 block truncate text-xs text-muted-foreground/80 italic">
              {conversation.excerpt}
            </span>
          )}
        </span>
      </button>

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon"
              aria-label={`Actions for ${conversationName(conversation.title)}`}
              className="absolute right-3 top-1.5 size-7 opacity-0 transition-opacity group-hover/row:opacity-100 data-[popup-open]:opacity-100 focus-visible:opacity-100"
            />
          }
        >
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => setEditing(true)}>
            <Pencil className="size-4" />
            Rename
          </DropdownMenuItem>
          {conversation.archived ? (
            <DropdownMenuItem onClick={() => onSetArchived(false)}>
              <ArchiveRestore className="size-4" />
              Unarchive
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => onSetArchived(true)}>
              <Archive className="size-4" />
              Archive
            </DropdownMenuItem>
          )}
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
  onSetArchived,
}: Props) {
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null);
  // Search text is view state, but the SEARCH ITSELF is not: the query goes to
  // Postgres, which is the only place that can look inside message bodies
  // without shipping every transcript to the browser.
  const query = useChatView((s) => s.search);
  const setQuery = useChatView((s) => s.setSearch);
  const showArchived = useChatView((s) => s.showArchived);
  const toggleArchived = useChatView((s) => s.toggleArchived);

  const searching = query.trim().length > 0;
  const active = useMemo(() => conversations.filter((c) => !c.archived), [conversations]);
  const archived = useMemo(() => conversations.filter((c) => c.archived), [conversations]);
  const archivedTotal = archived.length;

  // Searching reaches into the archive. Hiding an archived match would make the
  // archive a place things go to become unfindable, which is what deleting is
  // for.
  const archivedOpen = showArchived || searching;

  function rowFor(c: Conversation) {
    return (
      <Row
        key={c.id}
        conversation={c}
        showExcerpt={searching}
        selected={c.id === selectedId}
        onSelect={() => onSelect(c.id)}
        onRename={(title) => onRename(c.id, title)}
        onSetArchived={(a) => onSetArchived(c.id, a)}
        onAskDelete={() => setPendingDelete(c)}
      />
    );
  }

  return (
    <aside className="flex h-full w-72 shrink-0 flex-col border-r bg-card">
      <div className="flex shrink-0 items-center justify-between gap-2 px-4 py-3">
        <h2 className="text-sm font-medium">Conversations</h2>
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={onCreate}
            aria-label="New conversation"
          >
            <Plus className="size-4" />
          </Button>
          <SignOutMenu />
        </div>
      </div>

      <div className="shrink-0 px-3 pb-2">
        <div className="flex items-center gap-2 rounded-md border bg-background px-2.5 focus-within:ring-2 focus-within:ring-ring">
          <Search className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Escape" && setQuery("")}
            placeholder="Search conversations"
            aria-label="Search conversations"
            className="min-w-0 flex-1 bg-transparent py-1.5 text-sm outline-none placeholder:text-muted-foreground"
          />
          {searching && (
            <button
              type="button"
              onClick={() => setQuery("")}
              aria-label="Clear search"
              className="shrink-0 text-muted-foreground hover:text-foreground"
            >
              <X className="size-3.5" />
            </button>
          )}
        </div>
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
      ) : conversations.length === 0 && !searching ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
          <MessagesSquare className="size-7 text-muted-foreground" aria-hidden />
          <p className="text-sm text-muted-foreground">
            No conversations yet. Start one and it will be saved here — the
            transcript is kept by sparstrowgen, not by whichever agent answered.
          </p>
          <Button size="sm" onClick={onCreate}>
            <Plus className="size-4" />
            New conversation
          </Button>
        </div>
      ) : conversations.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
          <Search className="size-6 text-muted-foreground" aria-hidden />
          <p className="text-sm text-muted-foreground">
            Nothing matches &ldquo;{query.trim()}&rdquo; — titles, folders and
            message text were all searched, including the archive.
          </p>
        </div>
      ) : (
        <ScrollArea className="min-h-0 flex-1">
          <ul className="space-y-0.5 pb-3">{active.map(rowFor)}</ul>

          {archivedTotal > 0 && (
            <div className="pb-3">
              <button
                type="button"
                onClick={toggleArchived}
                aria-expanded={archivedOpen}
                className="flex w-full items-center gap-1.5 px-4 py-2 text-xs text-muted-foreground hover:text-foreground"
              >
                <ChevronRight
                  className={`size-3.5 transition-transform ${archivedOpen ? "rotate-90" : ""}`}
                  aria-hidden
                />
                Archived
                <span className="tabular-nums">
                  ({searching ? archived.length : archivedTotal})
                </span>
              </button>
              {archivedOpen && (
                <ul className="space-y-0.5">
                  {archived.length > 0 ? (
                    archived.map(rowFor)
                  ) : (
                    <li className="px-6 py-1 text-xs text-muted-foreground">
                      No archived conversation matches.
                    </li>
                  )}
                </ul>
              )}
            </div>
          )}
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
              &ldquo;{conversationName(pendingDelete?.title ?? "")}&rdquo; and its
              full transcript will
              be removed. Since the transcript lives here rather than with any
              agent, this cannot be recovered from the provider.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {/* Archiving is offered here because this dialog is the moment the
              intent is usually "get it out of my list", not "destroy it". */}
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            {!pendingDelete?.archived && (
              <Button
                variant="secondary"
                onClick={() => {
                  if (pendingDelete) onSetArchived(pendingDelete.id, true);
                  setPendingDelete(null);
                }}
              >
                <Archive className="size-4" />
                Archive instead
              </Button>
            )}
            <Button
              variant="destructive"
              onClick={() => {
                if (pendingDelete) onDelete(pendingDelete.id);
                setPendingDelete(null);
              }}
            >
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </aside>
  );
}
