"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FolderOpen, PlugZap, Plus } from "lucide-react";
import { toast } from "sonner";
import type {
  AgentMessage,
  Conversation,
  Entry,
  PendingSwitch,
  ProviderId,
} from "@/lib/chat-types";
import { mockConversations, mockProviders, mockReply } from "@/lib/chat.mock";
import { ConversationList } from "./conversation-list";
import { ProviderStrip } from "./provider-strip";
import {
  MessageList,
  MessageSkeleton,
  WorkingIndicator,
} from "./message-list";
import { Composer } from "./composer";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TooltipProvider } from "@/components/ui/tooltip";
import { formatTokens, formatUsd } from "./provider-meta";

/* Prototype state lives in useState. In the real app server state belongs to
   TanStack Query and only view state (draft, pending switch, filters) is local
   — see AGENTS.md §3 Hard constraints. Nothing here should be lifted as-is. */

type Surface = "ready" | "loading" | "offline";

/** Rough cost of catching a provider up. Real figure comes from the daemon. */
const TOKENS_PER_MESSAGE = 1300;

function now() {
  return new Date().toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function ChatSurface() {
  const [surface, setSurface] = useState<Surface>("ready");
  const [conversations, setConversations] =
    useState<Conversation[]>(mockConversations);
  const [selectedId, setSelectedId] = useState<string | null>("c1");
  const [pending, setPending] = useState<PendingSwitch | null>(null);
  const [draft, setDraft] = useState("");
  const [inFlight, setInFlight] = useState<{
    provider: ProviderId;
    model: string;
    elapsed: number;
    streams: boolean;
  } | null>(null);
  const [streamingId, setStreamingId] = useState<string | null>(null);

  const scrollRef = useRef<HTMLDivElement>(null);
  const timers = useRef<ReturnType<typeof setTimeout>[]>([]);

  useEffect(
    () => () => {
      timers.current.forEach(clearTimeout);
    },
    [],
  );

  const selected = useMemo(
    () => conversations.find((c) => c.id === selectedId) ?? null,
    [conversations, selectedId],
  );

  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      const el = scrollRef.current?.querySelector<HTMLElement>(
        "[data-slot='scroll-area-viewport']",
      );
      if (el) el.scrollTop = el.scrollHeight;
    });
  }, []);

  useEffect(scrollToBottom, [selected?.entries.length, scrollToBottom]);

  const offline = surface === "offline";

  function patch(id: string, fn: (c: Conversation) => Conversation) {
    setConversations((prev) => prev.map((c) => (c.id === id ? fn(c) : c)));
  }

  function handleCreate() {
    const id = `c${Date.now()}`;
    const fresh: Conversation = {
      id,
      title: "Untitled conversation",
      folder: "D:\\sparstrowgen",
      updated: "now",
      provider: "claude",
      model: "Sonnet 5",
      spendUsd: 0,
      tokens: 0,
      entries: [],
      seenBy: {},
    };
    setConversations((prev) => [fresh, ...prev]);
    setSelectedId(id);
    setPending(null);
    setDraft("");
  }

  function handleDelete(id: string) {
    const gone = conversations.find((c) => c.id === id);
    setConversations((prev) => prev.filter((c) => c.id !== id));
    if (selectedId === id) setSelectedId(null);
    toast("Conversation deleted", { description: gone?.title });
  }

  /** Selecting a provider is free. It records what a switch *would* cost and
   *  charges nothing until the next message is actually sent. */
  function handleSelectProvider(provider: ProviderId, model: string) {
    if (!selected) return;
    if (provider === selected.provider) {
      setPending(model === selected.model ? null : null);
      patch(selected.id, (c) => ({ ...c, model }));
      return;
    }
    const seen = selected.seenBy[provider] ?? 0;
    const gap = Math.max(selected.entries.length - seen, 0);
    if (gap === 0) {
      patch(selected.id, (c) => ({ ...c, provider, model }));
      setPending(null);
      return;
    }
    setPending({
      to: provider,
      toModel: model,
      messagesToReplay: gap,
      estimatedTokens: gap * TOKENS_PER_MESSAGE,
    });
  }

  function handleSend() {
    if (!selected || !draft.trim() || inFlight) return;
    const conversationId = selected.id;
    const text = draft.trim();
    setDraft("");

    const provider = pending?.to ?? selected.provider;
    const model = pending?.toModel ?? selected.model;
    const meta = mockProviders.find((p) => p.id === provider)!;
    const replay = pending;
    setPending(null);

    // The replay is written into the transcript here — the moment it is paid
    // for — not when the provider was picked.
    const added: Entry[] = [];
    if (replay) {
      added.push({
        id: `r${Date.now()}`,
        role: "replay",
        at: now(),
        to: replay.to,
        toModel: replay.toModel,
        messagesReplayed: replay.messagesToReplay,
        tokens: replay.estimatedTokens,
      });
    }
    added.push({ id: `u${Date.now()}`, role: "user", at: now(), text });

    patch(conversationId, (c) => ({
      ...c,
      provider,
      model,
      updated: "now",
      entries: [...c.entries, ...added],
      tokens: c.tokens + (replay?.estimatedTokens ?? 0),
    }));

    const replyId = `a${Date.now()}`;
    setInFlight({ provider, model, elapsed: 0, streams: meta.streams });

    const tick = setInterval(
      () => setInFlight((f) => (f ? { ...f, elapsed: f.elapsed + 1 } : f)),
      1000,
    );

    const finish = (full: string) => {
      clearInterval(tick);
      setInFlight(null);
      setStreamingId(null);
      patch(conversationId, (c) => {
        const usage = meta.reportsUsd
          ? { tokens: 12400, usd: 0.03 }
          : { tokens: 9800 };
        const next = c.entries.map((e) =>
          e.id === replyId
            ? ({ ...(e as AgentMessage), text: full, usage, code: mockReply.code } as Entry)
            : e,
        );
        const exists = c.entries.some((e) => e.id === replyId);
        const reply: AgentMessage = {
          id: replyId,
          role: "agent",
          at: now(),
          provider,
          model,
          text: full,
          usage,
          code: mockReply.code,
        };
        return {
          ...c,
          entries: exists ? next : [...c.entries, reply],
          tokens: c.tokens + usage.tokens,
          spendUsd: c.spendUsd + (usage.usd ?? 0),
          seenBy: { ...c.seenBy, [provider]: c.entries.length + 2 },
        };
      });
      scrollToBottom();
    };

    if (meta.streams) {
      // Incremental: agy is verified to send ~25-35 char text_delta chunks.
      patch(conversationId, (c) => ({
        ...c,
        entries: [
          ...c.entries,
          {
            id: replyId,
            role: "agent",
            at: now(),
            provider,
            model,
            text: "",
          } as AgentMessage,
        ],
      }));
      setStreamingId(replyId);
      const words = mockReply.text.split(" ");
      let i = 0;
      const step = () => {
        i += 3;
        const partial = words.slice(0, i).join(" ");
        patch(conversationId, (c) => ({
          ...c,
          entries: c.entries.map((e) =>
            e.id === replyId ? ({ ...(e as AgentMessage), text: partial } as Entry) : e,
          ),
        }));
        scrollToBottom();
        if (i < words.length) timers.current.push(setTimeout(step, 55));
        else finish(mockReply.text);
      };
      timers.current.push(setTimeout(step, 400));
    } else {
      // codex is VERIFIED to emit no deltas — nothing at all until the whole
      // answer lands. The working indicator is the only sign of life.
      timers.current.push(setTimeout(() => finish(mockReply.text), 4200));
    }
  }

  const activeProvider = selected?.provider ?? "claude";
  const activeModel = selected?.model ?? "Sonnet 5";

  return (
    <TooltipProvider>
      <div className="flex h-full">
        <ConversationList
          conversations={conversations}
          selectedId={selectedId}
          loading={surface === "loading"}
          onSelect={(id) => {
            setSelectedId(id);
            setPending(null);
            setDraft("");
          }}
          onCreate={handleCreate}
          onRename={(id, title) => patch(id, (c) => ({ ...c, title }))}
          onDelete={handleDelete}
        />

        <main className="flex min-w-0 flex-1 flex-col">
          <ProviderStrip
            providers={
              offline
                ? mockProviders.map((p) =>
                    p.availability === "blocked"
                      ? p
                      : {
                          ...p,
                          availability: "waitable" as const,
                          unavailableReason: "Machine unreachable",
                          headroom: null,
                        },
                  )
                : mockProviders
            }
          />

          {/* Prototype-only. Not part of the design — a way to reach every
              state without waiting for one to occur. */}
          <div className="flex shrink-0 items-center gap-1.5 border-b border-dashed px-3 py-1.5">
            <span className="mr-1 text-[11px] uppercase tracking-wide text-muted-foreground">
              prototype states
            </span>
            {(
              [
                ["ready", "Populated"],
                ["loading", "Loading"],
                ["offline", "Daemon offline"],
              ] as const
            ).map(([key, label]) => (
              <Button
                key={key}
                size="sm"
                variant={surface === key ? "secondary" : "ghost"}
                className="h-6 px-2 text-xs"
                onClick={() => setSurface(key)}
              >
                {label}
              </Button>
            ))}
            <Button
              size="sm"
              variant={conversations.length === 0 ? "secondary" : "ghost"}
              className="h-6 px-2 text-xs"
              onClick={() =>
                conversations.length === 0
                  ? setConversations(mockConversations)
                  : (setConversations([]), setSelectedId(null))
              }
            >
              Empty
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="h-6 px-2 text-xs"
              onClick={() => setSelectedId("c3")}
            >
              Failed turn
            </Button>
          </div>

          {selected ? (
            <>
              <header className="flex shrink-0 items-center gap-3 border-b px-6 py-3">
                <div className="min-w-0 flex-1">
                  <h1 className="truncate text-sm font-medium">
                    {selected.title}
                  </h1>
                  <p className="mt-0.5 flex items-center gap-1.5 truncate text-xs text-muted-foreground">
                    <FolderOpen className="size-3.5 shrink-0" aria-hidden />
                    {selected.folder}
                  </p>
                </div>
                <div className="shrink-0 text-right font-mono text-xs text-muted-foreground">
                  <div>{formatUsd(selected.spendUsd)}</div>
                  <div>{formatTokens(selected.tokens)} tokens</div>
                </div>
              </header>

              <ScrollArea ref={scrollRef} className="min-h-0 flex-1">
                <div className="mx-auto max-w-3xl px-6 py-8">
                  {surface === "loading" ? (
                    <MessageSkeleton />
                  ) : selected.entries.length === 0 ? (
                    <div className="flex flex-col items-center gap-2 py-24 text-center">
                      <p className="text-sm font-medium">
                        Nothing said here yet
                      </p>
                      <p className="max-w-sm text-sm text-muted-foreground">
                        Ask {selected.provider} something about{" "}
                        {selected.folder.split("\\").pop()}. You can move this
                        conversation to another agent at any point without
                        losing it.
                      </p>
                    </div>
                  ) : (
                    <>
                      <MessageList
                        entries={selected.entries}
                        streamingId={streamingId}
                      />
                      {inFlight && (
                        <div className="mt-8">
                          <WorkingIndicator
                            provider={inFlight.provider}
                            model={inFlight.model}
                            elapsed={inFlight.elapsed}
                            streams={inFlight.streams}
                          />
                        </div>
                      )}
                    </>
                  )}
                </div>
              </ScrollArea>

              <Composer
                providers={mockProviders}
                activeProvider={activeProvider}
                activeModel={activeModel}
                pending={pending}
                disabled={offline || surface === "loading"}
                disabledReason={
                  offline
                    ? "Your machine is unreachable, so nothing new can be sent. Everything already said stays readable."
                    : undefined
                }
                value={draft}
                onChange={setDraft}
                onSelect={handleSelectProvider}
                onCancelSwitch={() => setPending(null)}
                onSend={handleSend}
              />
            </>
          ) : (
            <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
              <PlugZap className="size-8 text-muted-foreground" aria-hidden />
              <p className="text-sm font-medium">No conversation selected</p>
              <p className="max-w-sm text-sm text-muted-foreground">
                Pick one on the left, or start a new one. Whichever agent
                answers, the transcript is kept here.
              </p>
              <Button size="sm" onClick={handleCreate}>
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
