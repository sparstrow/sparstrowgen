"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FolderOpen, PlugZap, Plus, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import type { Model, ProviderId } from "@/lib/chat-types";
import { api } from "@/lib/api";
import {
  useArchiveConversation,
  useConversation,
  useConversations,
  useCreateConversation,
  useDeleteConversation,
  useProviders,
  useRealtime,
  useRenameConversation,
  useSendMessage,
} from "@/lib/queries";
import { useChatView } from "@/lib/store";
import { ConversationList } from "./conversation-list";
import { ProviderStrip } from "./provider-strip";
import { MessageList, MessageSkeleton, WorkingIndicator } from "./message-list";
import { Composer } from "./composer";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TooltipProvider } from "@/components/ui/tooltip";
import { formatTokens, formatUsd } from "./provider-meta";

/* Server state is TanStack Query's; view state is Zustand's; websocket events
   patch the Query cache and are never mirrored into the store (AGENTS.md §3).
   The only useState left is the wall clock that drives the elapsed counter and
   the daemon-reachable flag, neither of which is server data. */

export function ChatSurface() {
  const selectedId = useChatView((s) => s.selectedId);
  const select = useChatView((s) => s.select);
  const search = useChatView((s) => s.search);
  const pending = useChatView((s) => s.pending);
  const setPending = useChatView((s) => s.setPending);
  const inFlight = useChatView((s) => s.inFlight);
  const setInFlight = useChatView((s) => s.setInFlight);
  const drafts = useChatView((s) => s.drafts);
  const setDraft = useChatView((s) => s.setDraft);
  const clearDraft = useChatView((s) => s.clearDraft);

  const [daemonOnline, setDaemonOnline] = useState(false);
  const [now, setNow] = useState(() => Date.now());

  const onDaemon = useCallback((online: boolean) => setDaemonOnline(online), []);
  useRealtime(onDaemon);

  const { data: providers = [] } = useProviders();
  const conversations = useConversations(search);
  const conversation = useConversation(selectedId);

  const create = useCreateConversation();
  const rename = useRenameConversation();
  const archive = useArchiveConversation();
  const remove = useDeleteConversation();
  const send = useSendMessage();

  const selected = conversation.data ?? null;
  const draft = selectedId ? (drafts[selectedId] ?? "") : "";

  const scrollRef = useRef<HTMLDivElement>(null);
  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      const el = scrollRef.current?.querySelector<HTMLElement>(
        "[data-slot='scroll-area-viewport']",
      );
      if (el) el.scrollTop = el.scrollHeight;
    });
  }, []);
  useEffect(scrollToBottom, [selected?.entries.length, selectedId, scrollToBottom]);

  // Open the first conversation once, so the app does not start on an empty
  // pane when there is something to read.
  const list = conversations.data;
  useEffect(() => {
    if (selectedId || !list?.length || search) return;
    const first = list.find((c) => !c.archived);
    if (first) select(first.id);
  }, [list, selectedId, search, select]);

  // A turn in flight needs a ticking clock: on codex nothing else moves until
  // the whole answer lands. The clock only runs while one is in flight, and
  // elapsed is derived from it rather than accumulated, so there is nothing to
  // reset when the turn ends.
  useEffect(() => {
    if (!inFlight) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [inFlight]);
  const elapsed = inFlight
    ? Math.max(0, Math.floor((now - inFlight.startedAt) / 1000))
    : 0;

  // The turn is over when its entry stops being empty or reports a failure.
  useEffect(() => {
    if (!inFlight || !selected) return;
    const entry = selected.entries.find((e) => e.id === inFlight.entryId);
    if (entry && entry.role === "agent" && (entry.text || entry.failure)) {
      const stillStreaming = !entry.usage && !entry.failure;
      if (!stillStreaming) setInFlight(null);
    }
  }, [selected, inFlight, setInFlight]);

  const activeProvider: ProviderId =
    selected?.provider ?? providers[0]?.id ?? "claude";
  const activeModel: Model =
    selected?.model ?? providers[0]?.model ?? { id: "", label: "—" };

  const usableProviders = useMemo(
    () =>
      providers.map((p) =>
        daemonOnline
          ? p
          : {
              ...p,
              availability: "waitable" as const,
              unavailableReason: "Machine unreachable",
              headroom: null,
            },
      ),
    [providers, daemonOnline],
  );

  // -------------------------------------------------------------------------

  async function handleCreate() {
    const created = await create.mutateAsync({
      provider: activeProvider,
      model: activeModel,
    });
    select(created.id);
  }

  /** Selecting a provider is free and writes nothing. The server quotes what a
   *  catch-up would cost; it is charged on the next message, not now. */
  async function handleSelectProvider(provider: ProviderId, model: Model) {
    if (!selected) return;
    if (provider === selected.provider) {
      // Changing model within a provider replays nothing: the provider's own
      // session carries over and seenBy is keyed by provider, not by model.
      setPending(null);
      setPendingModelLocally(model);
      return;
    }
    try {
      const cost = await api.switchCost(selected.id, provider);
      if (cost.messagesToReplay === 0) {
        setPending({
          to: provider,
          toModel: model,
          messagesToReplay: 0,
          estimatedTokens: 0,
        });
        return;
      }
      setPending({
        to: provider,
        toModel: model,
        messagesToReplay: cost.messagesToReplay,
        estimatedTokens: cost.estimatedTokens,
      });
    } catch (err) {
      toast.error("Could not price that switch", {
        description: (err as Error).message,
      });
    }
  }

  /** A model change within the current provider is free and immediate, so it is
   *  carried as a zero-cost pending switch rather than written to the server —
   *  the conversation's stored model only changes when a message is sent. */
  function setPendingModelLocally(model: Model) {
    if (!selected) return;
    if (model.id === selected.model.id) return;
    setPending({
      to: selected.provider,
      toModel: model,
      messagesToReplay: 0,
      estimatedTokens: 0,
    });
  }

  async function handleSend() {
    if (!selected || !draft.trim() || inFlight) return;
    const text = draft.trim();
    const provider = pending?.to ?? selected.provider;
    const model = pending?.toModel ?? selected.model;
    clearDraft(selected.id);
    setPending(null);

    try {
      const { entry } = await send.mutateAsync({ id: selected.id, text, provider, model });
      setNow(Date.now());
      setInFlight({ entryId: entry.id, provider, model, startedAt: Date.now() });
    } catch (err) {
      setDraft(selected.id, text); // give it back rather than losing what was typed
      toast.error("Message not sent", { description: (err as Error).message });
    }
  }

  // -------------------------------------------------------------------------

  const streamingId = inFlight?.entryId ?? null;
  const visibleEntries = useMemo(
    () =>
      (selected?.entries ?? []).filter(
        // The placeholder an in-flight turn streams into is empty until text
        // arrives; the working indicator speaks for it until then.
        (e) => !(e.role === "agent" && e.id === streamingId && !e.text),
      ),
    [selected?.entries, streamingId],
  );

  const inFlightProvider = providers.find((p) => p.id === inFlight?.provider);

  if (conversations.isError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
        <PlugZap className="size-8 text-muted-foreground" aria-hidden />
        <p className="text-sm font-medium">Can&apos;t reach the sparstrowgen server</p>
        <p className="max-w-md text-sm text-muted-foreground">
          {(conversations.error as Error).message}
        </p>
        <p className="max-w-md text-xs text-muted-foreground">
          Start it with <code className="font-mono">make server</code>, and check
          Postgres is up with <code className="font-mono">make db</code>.
        </p>
        <Button size="sm" onClick={() => void conversations.refetch()}>
          <RefreshCw className="size-4" />
          Try again
        </Button>
      </div>
    );
  }

  return (
    <TooltipProvider>
      <div className="flex h-full">
        <ConversationList
          conversations={conversations.data ?? []}
          selectedId={selectedId}
          loading={conversations.isPending}
          onSelect={select}
          onCreate={() => void handleCreate()}
          onRename={(id, title) => rename.mutate({ id, patch: { title } })}
          onDelete={(id) => {
            remove.mutate(id);
            if (selectedId === id) select(null);
          }}
          onSetArchived={(id, archived) => {
            archive.mutate({ id, patch: { archived } });
            toast(archived ? "Conversation archived" : "Conversation restored", {
              action: {
                label: "Undo",
                onClick: () =>
                  archive.mutate({ id, patch: { archived: !archived } }),
              },
            });
          }}
        />

        <main className="flex min-w-0 flex-1 flex-col">
          <ProviderStrip providers={usableProviders} />

          {selected ? (
            <>
              <header className="flex shrink-0 items-center gap-3 border-b px-6 py-3">
                <div className="min-w-0 flex-1">
                  <h1 className="truncate text-sm font-medium">{selected.title}</h1>
                  <p className="mt-0.5 flex items-center gap-1.5 truncate text-xs text-muted-foreground">
                    <FolderOpen className="size-3.5 shrink-0" aria-hidden />
                    {selected.folder}
                  </p>
                </div>
                <div className="shrink-0 text-right font-mono text-xs text-muted-foreground">
                  {/* Only claude states a dollar figure, so zero spend on a
                      conversation answered by the others means "not reported",
                      not "free". Showing it only when there is one keeps that
                      honest. */}
                  {selected.spendUsd > 0 && <div>{formatUsd(selected.spendUsd)}</div>}
                  <div>{formatTokens(selected.tokens)} tokens</div>
                </div>
              </header>

              <ScrollArea ref={scrollRef} className="min-h-0 flex-1">
                <div className="mx-auto max-w-3xl px-6 py-8">
                  {conversation.isPending ? (
                    <MessageSkeleton />
                  ) : visibleEntries.length === 0 && !inFlight ? (
                    <div className="flex flex-col items-center gap-2 py-24 text-center">
                      <p className="text-sm font-medium">Nothing said here yet</p>
                      <p className="max-w-sm text-sm text-muted-foreground">
                        Ask {selected.provider} something about{" "}
                        {selected.folder.split(/[\\/]/).pop()}. You can move this
                        conversation to another agent at any point without losing
                        it.
                      </p>
                    </div>
                  ) : (
                    <>
                      <MessageList entries={visibleEntries} streamingId={streamingId} />
                      {inFlight && (
                        <div className="mt-8">
                          <WorkingIndicator
                            provider={inFlight.provider}
                            model={inFlight.model}
                            elapsed={elapsed}
                            streams={inFlightProvider?.streams ?? false}
                          />
                        </div>
                      )}
                    </>
                  )}
                </div>
              </ScrollArea>

              <Composer
                providers={usableProviders}
                activeProvider={activeProvider}
                activeModel={activeModel}
                pending={pending}
                disabled={!daemonOnline || inFlight !== null}
                disabledReason={
                  !daemonOnline
                    ? "Your machine is unreachable, so nothing new can be sent. Everything already said stays readable."
                    : undefined
                }
                value={draft}
                onChange={(v) => selectedId && setDraft(selectedId, v)}
                onSelect={(p, m) => void handleSelectProvider(p, m)}
                onCancelSwitch={() => setPending(null)}
                onSend={() => void handleSend()}
              />
            </>
          ) : (
            <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
              <PlugZap className="size-8 text-muted-foreground" aria-hidden />
              <p className="text-sm font-medium">No conversation selected</p>
              <p className="max-w-sm text-sm text-muted-foreground">
                Pick one on the left, or start a new one. Whichever agent answers,
                the transcript is kept here.
              </p>
              <Button size="sm" onClick={() => void handleCreate()}>
                <Plus className="size-4" />
                New conversation
              </Button>
            </div>
          )}
        </main>
      </div>
    </TooltipProvider>
  );
}
