"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";
import { api, connect, type ServerEvent } from "./api";
import type { Conversation, Entry, Model, Provider, ProviderId } from "./chat-types";

/* Every read of server state goes through here, and every realtime event
   patches this cache rather than a second copy of the data (AGENTS.md §3). */

export const keys = {
  providers: ["providers"] as const,
  conversations: (q: string) => ["conversations", q] as const,
  conversation: (id: string) => ["conversation", id] as const,
};

export function useProviders() {
  return useQuery({
    queryKey: keys.providers,
    queryFn: api.providers,
    // Providers arrive over the websocket the moment the daemon reports them,
    // so polling would only duplicate a push we already get.
    staleTime: Infinity,
    initialData: [] as Provider[],
  });
}

export function useConversations(search: string) {
  return useQuery({
    queryKey: keys.conversations(search),
    queryFn: () => api.conversations(search),
    // Keeps the previous list on screen while a new search resolves, instead of
    // flashing empty on every keystroke.
    placeholderData: (prev) => prev,
  });
}

export function useConversation(id: string | null) {
  return useQuery({
    queryKey: keys.conversation(id ?? ""),
    queryFn: () => api.conversation(id!),
    enabled: id !== null,
  });
}

// ---------------------------------------------------------------------------
// mutations
// ---------------------------------------------------------------------------

/** Invalidate every conversation list, whatever search term keyed it. */
function invalidateLists(qc: QueryClient) {
  return qc.invalidateQueries({ queryKey: ["conversations"] });
}

export function useCreateConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { provider?: ProviderId; model?: Model; folder?: string }) =>
      api.create(input),
    onSuccess: (created) => {
      qc.setQueryData(keys.conversation(created.id), created);
      void invalidateLists(qc);
    },
    onError: (err: Error) =>
      toast.error("Could not start a conversation", { description: err.message }),
  });
}

/* Rename and archive are optimistic: the outcome is predictable, the user stays
   where they are, failure is rare, and rollback is one cache write — which is
   the whole test AGENTS.md §3 sets for doing it. Sending a message is not
   optimistic, because what comes back is a model's answer and nothing about it
   is predictable. */
function useOptimisticPatch<T extends { title?: string; archived?: boolean }>(
  apply: (c: Conversation, patch: T) => Conversation,
  failure: string,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: T }) => api.patch(id, patch),
    onMutate: async ({ id, patch }) => {
      await qc.cancelQueries({ queryKey: ["conversations"] });
      const lists = qc.getQueriesData<Conversation[]>({ queryKey: ["conversations"] });
      const detail = qc.getQueryData<Conversation>(keys.conversation(id));

      for (const [key, list] of lists) {
        if (!list) continue;
        qc.setQueryData(
          key,
          list.map((c) => (c.id === id ? apply(c, patch) : c)),
        );
      }
      if (detail) qc.setQueryData(keys.conversation(id), apply(detail, patch));

      return { lists, detail, id };
    },
    onError: (err: Error, _vars, ctx) => {
      // Put back exactly what was there, rather than refetching and hoping.
      for (const [key, list] of ctx?.lists ?? []) qc.setQueryData(key, list);
      if (ctx?.detail) qc.setQueryData(keys.conversation(ctx.id), ctx.detail);
      toast.error(failure, { description: err.message });
    },
    onSettled: () => invalidateLists(qc),
  });
}

export function useRenameConversation() {
  return useOptimisticPatch<{ title: string }>(
    (c, p) => ({ ...c, title: p.title }),
    "Rename failed",
  );
}

export function useArchiveConversation() {
  return useOptimisticPatch<{ archived: boolean }>(
    (c, p) => ({ ...c, archived: p.archived }),
    "Could not archive",
  );
}

export function useDeleteConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.remove(id),
    onSuccess: (_v, id) => {
      qc.removeQueries({ queryKey: keys.conversation(id) });
      void invalidateLists(qc);
    },
    onError: (err: Error) =>
      toast.error("Could not delete", { description: err.message }),
  });
}

export function useSendMessage() {
  return useMutation({
    mutationFn: ({
      id,
      text,
      provider,
      model,
    }: {
      id: string;
      text: string;
      provider: ProviderId;
      model: Model;
    }) => api.send(id, { text, provider, model }),
  });
}

// ---------------------------------------------------------------------------
// realtime
// ---------------------------------------------------------------------------

/** Patches the Query cache from websocket events.
 *
 *  Entries are patched in place rather than triggering a refetch: a delta
 *  arrives every few characters, and refetching a whole transcript on each one
 *  would be absurd. Conversation-level changes invalidate, because they are
 *  rare and the server's version is authoritative. */
export function useRealtime(onDaemon: (online: boolean) => void) {
  const qc = useQueryClient();

  useEffect(() => {
    const patchEntries = (
      conversationId: string,
      fn: (entries: Entry[]) => Entry[],
    ) =>
      qc.setQueryData<Conversation>(keys.conversation(conversationId), (prev) =>
        prev ? { ...prev, entries: fn(prev.entries) } : prev,
      );

    const handle = (ev: ServerEvent) => {
      switch (ev.type) {
        case "providers":
          qc.setQueryData(keys.providers, ev.providers);
          break;

        case "daemon":
          onDaemon(ev.online);
          break;

        case "conversation":
          qc.setQueryData<Conversation>(
            keys.conversation(ev.conversation.id),
            (prev) =>
              // The list payload carries no entries. Keeping the open
              // transcript is what stops a usage update blanking the screen.
              prev
                ? { ...ev.conversation, entries: prev.entries, seenBy: prev.seenBy }
                : prev,
          );
          void invalidateLists(qc);
          break;

        case "entry_added":
          patchEntries(ev.conversationId, (entries) =>
            entries.some((e) => e.id === ev.entry.id)
              ? entries
              : [...entries, ev.entry],
          );
          break;

        case "entry_delta":
          patchEntries(ev.conversationId, (entries) =>
            entries.map((e) =>
              e.id === ev.entryId && e.role === "agent"
                ? ({ ...e, text: (e.text ?? "") + ev.text } as Entry)
                : e,
            ),
          );
          break;

        case "entry_done":
          patchEntries(ev.conversationId, (entries) =>
            entries.map((e) => (e.id === ev.entry.id ? ev.entry : e)),
          );
          qc.setQueryData<Conversation>(
            keys.conversation(ev.conversationId),
            (prev) =>
              prev
                ? {
                    ...prev,
                    // The provider has now seen everything up to its own reply,
                    // so a switch away and back replays only what follows.
                    seenBy: { ...prev.seenBy, [ev.entry.provider]: prev.entries.length },
                  }
                : prev,
          );
          if (ev.entry.failure) {
            toast.error("Turn did not finish", { description: ev.entry.failure });
          }
          void invalidateLists(qc);
          break;
      }
    };

    return connect(handle, (open) => {
      // The socket being up says the server is reachable. Whether the owner's
      // machine is reachable is a separate fact the server tells us.
      if (!open) onDaemon(false);
    });
  }, [qc, onDaemon]);
}
