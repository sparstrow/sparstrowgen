"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FolderOpen, PlugZap, Plus, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import type {
  Conversation,
  Entry,
  Model,
  PendingSwitch,
  Provider,
  ProviderId,
} from "@/lib/chat-types";
import { api, connect, type ServerEvent } from "@/lib/api";
import { ConversationList } from "./conversation-list";
import { ProviderStrip } from "./provider-strip";
import { MessageList, MessageSkeleton, WorkingIndicator } from "./message-list";
import { Composer } from "./composer";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TooltipProvider } from "@/components/ui/tooltip";
import { formatTokens, formatUsd } from "./provider-meta";

/* View state lives here; server state comes from the API and is patched by
   websocket events. TanStack Query is the eventual home for the second half
   (AGENTS.md §3) — see docs/KnownGaps.md G-9 for what that costs today. */

export function ChatSurface() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selected, setSelected] = useState<Conversation | null>(null);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [daemonOnline, setDaemonOnline] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pending, setPending] = useState<PendingSwitch | null>(null);
  const [draft, setDraft] = useState("");
  const [inFlight, setInFlight] = useState<{
    entryId: string;
    provider: ProviderId;
    model: Model;
    elapsed: number;
  } | null>(null);

  const scrollRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      const el = scrollRef.current?.querySelector<HTMLElement>(
        "[data-slot='scroll-area-viewport']",
      );
      if (el) el.scrollTop = el.scrollHeight;
    });
  }, []);

  const refreshList = useCallback(async () => {
    const list = await api.conversations();
    setConversations(list);
    return list;
  }, []);

  const openConversation = useCallback(
    async (id: string) => {
      setPending(null);
      setDraft("");
      const full = await api.conversation(id);
      setSelected(full);
      scrollToBottom();
    },
    [scrollToBottom],
  );

  // Initial load.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [list, ps] = await Promise.all([api.conversations(), api.providers()]);
        if (cancelled) return;
        setConversations(list);
        setProviders(ps);
        const first = list.find((c) => !c.archived);
        if (first) setSelected(await api.conversation(first.id));
        setLoadError(null);
      } catch (err) {
        if (!cancelled) setLoadError((err as Error).message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Live feed.
  useEffect(() => {
    const patchSelected = (id: string, fn: (c: Conversation) => Conversation) =>
      setSelected((cur) => (cur && cur.id === id ? fn(cur) : cur));

    const handle = (ev: ServerEvent) => {
      switch (ev.type) {
        case "providers":
          setProviders(ev.providers);
          break;

        case "daemon":
          setDaemonOnline(ev.online);
          break;

        case "conversation": {
          const next = ev.conversation;
          setConversations((prev) => {
            const without = prev.filter((c) => c.id !== next.id);
            return [next, ...without];
          });
          // The list payload carries no entries; keeping the open transcript is
          // what stops a usage update from blanking the conversation on screen.
          patchSelected(next.id, (c) => ({ ...next, entries: c.entries, seenBy: c.seenBy }));
          break;
        }

        case "entry_added":
          patchSelected(ev.conversationId, (c) =>
            c.entries.some((e) => e.id === ev.entry.id)
              ? c
              : { ...c, entries: [...c.entries, ev.entry] },
          );
          scrollToBottom();
          break;

        case "entry_delta":
          patchSelected(ev.conversationId, (c) => ({
            ...c,
            entries: c.entries.map((e) =>
              e.id === ev.entryId && e.role === "agent"
                ? ({ ...e, text: (e.text ?? "") + ev.text } as Entry)
                : e,
            ),
          }));
          scrollToBottom();
          break;

        case "entry_done":
          patchSelected(ev.conversationId, (c) => ({
            ...c,
            entries: c.entries.map((e) => (e.id === ev.entry.id ? ev.entry : e)),
            // The provider has now seen everything, so a switch away and back
            // replays only what comes after this point.
            seenBy: { ...c.seenBy, [ev.entry.provider as ProviderId]: c.entries.length },
          }));
          setInFlight((f) => (f && f.entryId === ev.entry.id ? null : f));
          if (ev.entry.failure) {
            toast.error("Turn did not finish", { description: ev.entry.failure });
          }
          void refreshList();
          scrollToBottom();
          break;
      }
    };

    return connect(handle, (online) => {
      // The socket being up says the server is reachable. Whether the owner's
      // machine is reachable is a separate fact, and the server tells us.
      if (!online) setDaemonOnline(false);
    });
  }, [refreshList, scrollToBottom]);

  // A turn in flight needs a ticking clock, because on codex nothing else moves
  // until the whole answer lands.
  useEffect(() => {
    if (!inFlight) return;
    const t = setInterval(
      () => setInFlight((f) => (f ? { ...f, elapsed: f.elapsed + 1 } : f)),
      1000,
    );
    return () => clearInterval(t);
  }, [inFlight?.entryId]); // eslint-disable-line react-hooks/exhaustive-deps

  const activeProvider: ProviderId = selected?.provider ?? providers[0]?.id ?? "claude";
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
    try {
      const created = await api.create({
        provider: activeProvider,
        model: activeModel,
      });
      setConversations((prev) => [created, ...prev]);
      setSelected({ ...created, entries: [], seenBy: {} });
      setPending(null);
      setDraft("");
    } catch (err) {
      toast.error("Could not start a conversation", {
        description: (err as Error).message,
      });
    }
  }

  async function handleRename(id: string, title: string) {
    setConversations((prev) => prev.map((c) => (c.id === id ? { ...c, title } : c)));
    try {
      await api.patch(id, { title });
    } catch (err) {
      toast.error("Rename failed", { description: (err as Error).message });
      void refreshList();
    }
  }

  async function handleSetArchived(id: string, archived: boolean) {
    const before = conversations.find((c) => c.id === id);
    setConversations((prev) => prev.map((c) => (c.id === id ? { ...c, archived } : c)));
    try {
      await api.patch(id, { archived });
      toast(archived ? "Conversation archived" : "Conversation restored", {
        description: before?.title,
        action: {
          label: "Undo",
          onClick: () => void handleSetArchived(id, !archived),
        },
      });
    } catch (err) {
      toast.error("Could not archive", { description: (err as Error).message });
      void refreshList();
    }
  }

  async function handleDelete(id: string) {
    const gone = conversations.find((c) => c.id === id);
    try {
      await api.remove(id);
      setConversations((prev) => prev.filter((c) => c.id !== id));
      if (selected?.id === id) setSelected(null);
      toast("Conversation deleted", { description: gone?.title });
    } catch (err) {
      toast.error("Could not delete", { description: (err as Error).message });
    }
  }

  /** Selecting a provider is free and writes nothing. The server quotes what a
   *  catch-up would cost; it is charged on the next message, not now. */
  async function handleSelectProvider(provider: ProviderId, model: Model) {
    if (!selected) return;
    if (provider === selected.provider) {
      // Changing model within a provider replays nothing: the provider's own
      // session carries over and seenBy is keyed by provider, not by model.
      setPending(null);
      setSelected({ ...selected, model });
      return;
    }
    try {
      const cost = await api.switchCost(selected.id, provider);
      if (cost.messagesToReplay === 0) {
        setPending(null);
        setSelected({ ...selected, provider, model });
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

  async function handleSend() {
    if (!selected || !draft.trim() || inFlight) return;
    const text = draft.trim();
    const provider = pending?.to ?? selected.provider;
    const model = pending?.toModel ?? selected.model;
    setDraft("");
    setPending(null);

    try {
      const { entry } = await api.send(selected.id, { text, provider, model });
      setSelected((c) => (c ? { ...c, provider, model } : c));
      setInFlight({ entryId: entry.id, provider, model, elapsed: 0 });
    } catch (err) {
      setDraft(text); // give it back rather than losing what was typed
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

  const provider = providers.find((p) => p.id === inFlight?.provider);

  if (loadError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
        <PlugZap className="size-8 text-muted-foreground" aria-hidden />
        <p className="text-sm font-medium">Can&apos;t reach the sparstrowgen server</p>
        <p className="max-w-md text-sm text-muted-foreground">{loadError}</p>
        <p className="max-w-md text-xs text-muted-foreground">
          Start it with <code className="font-mono">go run ./cmd/server</code> in{" "}
          <code className="font-mono">server/</code>, and check Postgres is up with{" "}
          <code className="font-mono">docker compose up -d</code>.
        </p>
        <Button size="sm" onClick={() => location.reload()}>
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
          conversations={conversations}
          selectedId={selected?.id ?? null}
          loading={loading}
          onSelect={(id) => void openConversation(id)}
          onCreate={() => void handleCreate()}
          onRename={(id, title) => void handleRename(id, title)}
          onDelete={(id) => void handleDelete(id)}
          onSetArchived={(id, a) => void handleSetArchived(id, a)}
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
                  {loading ? (
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
                            elapsed={inFlight.elapsed}
                            streams={provider?.streams ?? false}
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
                disabled={!daemonOnline || loading || inFlight !== null}
                disabledReason={
                  !daemonOnline
                    ? "Your machine is unreachable, so nothing new can be sent. Everything already said stays readable."
                    : undefined
                }
                value={draft}
                onChange={setDraft}
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
