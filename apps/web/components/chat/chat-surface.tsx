"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FolderOpen, Pencil, PlugZap, Plus, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import type { Model, ProviderId } from "@/lib/chat-types";
import { api } from "@/lib/api";
import {
  useArchiveConversation,
  useConversation,
  useConversations,
  useCreateConversation,
  useDaemon,
  useDeleteConversation,
  useMachines,
  useProviders,
  useRenameConversation,
  useSendMessage,
  useSetFolder,
  useStopTurn,
  useWorkspaceReach,
} from "@/lib/queries";
import { useChatView, type TranscriptView } from "@/lib/store";
import { ConversationList } from "./conversation-list";
import { ConversationName } from "./conversation-name";
import { ProviderStrip } from "./provider-strip";
import { MessageList, MessageSkeleton } from "./message-list";
import { WorkingIndicator } from "./working-indicator";
import { RawTranscript } from "./raw-transcript";
import { ChoiceToggle } from "./choice-toggle";
import { FolderPicker } from "./folder-picker";
import { Composer } from "./composer";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppHeader, AppShell, PaneHeader } from "@/components/shell/app-shell";
import { SetupCard } from "@/components/setup/setup-card";
import { useIsMobile } from "@/hooks/use-mobile";
import { formatTokens, formatUsd } from "./provider-meta";

/* Server state is TanStack Query's; view state is Zustand's; websocket events
   patch the Query cache and are never mirrored into the store (AGENTS.md §3).
   The only useState left is whether the folder picker is open, which is not
   server data either. The working indicator keeps its own clock. */

const VIEWS: { id: TranscriptView; label: string }[] = [
  { id: "rendered", label: "Rendered" },
  { id: "raw", label: "Raw" },
];

export function ChatSurface() {
  const selectedId = useChatView((s) => s.selectedId);
  const select = useChatView((s) => s.select);
  const search = useChatView((s) => s.search);
  const pending = useChatView((s) => s.pending);
  const setPending = useChatView((s) => s.setPending);
  const setInFlight = useChatView((s) => s.setInFlight);
  // Only the open conversation's turn: one running elsewhere must not lock or
  // show as working here (docs/Bugs.md B-29).
  const inFlight = useChatView((s) =>
    s.selectedId ? (s.inFlight[s.selectedId] ?? null) : null,
  );
  const transcriptView = useChatView((s) => s.transcriptView);
  const setTranscriptView = useChatView((s) => s.setTranscriptView);
  const drafts = useChatView((s) => s.drafts);
  const setDraft = useChatView((s) => s.setDraft);
  const clearDraft = useChatView((s) => s.clearDraft);

  const [pickingFolder, setPickingFolder] = useState(false);

  // Whether the computer can be reached, and whether it is too old to be sent
  // work (spec US3). Both are the server's to report, so they come from the
  // Query cache the socket patches rather than from local state — the header
  // shows them on every page now, not only here.
  const { online: daemonOnline, tooOld: daemonTooOld } = useDaemon();

  // "Unreachable" and "you have never connected one" are different facts and
  // must not share a sentence (docs/Bugs.md B-46). Only once the list has
  // actually loaded, because `[]` while pending would tell a person with a
  // computer that they have none.
  const machines = useMachines();
  const noComputerYet = machines.isSuccess && (machines.data?.length ?? 0) === 0;

  // Whether THIS workspace can run anything — not whether any computer on the
  // account is up, which is the question the composer asked until workspaces
  // got computers of their own (docs/Bugs.md B-49). The account's answer stands
  // in only until the workspace's list has loaded.
  const reach = useWorkspaceReach();
  const canReach = reach.known ? reach.reachable : daemonOnline;
  const noComputerInWorkspace =
    !noComputerYet && reach.known && reach.assigned.length === 0 && !reach.reachable;

  // On a phone the list and the conversation are two screens, so having one
  // open is what "show the conversation" means.
  const isMobile = useIsMobile();

  const { data: providers = [] } = useProviders();
  const conversations = useConversations(search);
  const conversation = useConversation(selectedId);

  const create = useCreateConversation();
  const rename = useRenameConversation();
  const archive = useArchiveConversation();
  const remove = useDeleteConversation();
  const setFolder = useSetFolder();
  const send = useSendMessage();
  const stop = useStopTurn();

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
  // pane when there is something to read. Not on a phone: there the list is a
  // screen of its own, and opening something would skip straight past it.
  const list = conversations.data;
  useEffect(() => {
    if (isMobile || selectedId || !list?.length || search) return;
    const first = list.find((c) => !c.archived);
    if (first) select(first.id);
  }, [list, selectedId, search, select, isMobile]);

  // The turn is over when its entry says so: usage means it finished, a failure
  // means it broke, and stopped means it was called back.
  //
  // `stopped` has to be checked on its own rather than folded into "has text or
  // has a failure". A stopped codex turn has neither — codex sends nothing at
  // all until the whole answer is ready — so without this the working indicator
  // would tick forever on the one provider where stopping helps most.
  useEffect(() => {
    if (!inFlight || !selected) return;
    const entry = selected.entries.find((e) => e.id === inFlight.entryId);
    if (!entry || entry.role !== "agent") return;
    if (entry.stopped || entry.failure || entry.usage) setInFlight(selected.id, null);
  }, [selected, inFlight, setInFlight]);

  const activeProvider: ProviderId =
    selected?.provider ?? providers[0]?.id ?? "claude";
  const activeModel: Model =
    selected?.model ?? providers[0]?.model ?? { id: "", label: "—" };

  const usableProviders = useMemo(
    () =>
      providers.map((p) =>
        !canReach
          ? {
              ...p,
              availability: "waitable" as const,
              unavailableReason: "Machine unreachable",
              headroom: null,
            }
          : daemonTooOld
            ? {
                ...p,
                availability: "blocked" as const,
                unavailableReason: "Computer needs an update",
              }
            : p,
      ),
    [providers, canReach, daemonTooOld],
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
    const conversationId = selected.id;
    clearDraft(conversationId);
    setPending(null);

    try {
      const { entry } = await send.mutateAsync({ id: conversationId, text, provider, model });
      // Against the conversation it was sent from, which may no longer be the
      // open one by the time the server answers.
      setInFlight(conversationId, { entryId: entry.id, provider, model, startedAt: Date.now() });
    } catch (err) {
      setDraft(conversationId, text); // give it back rather than losing what was typed
      toast.error("Message not sent", { description: (err as Error).message });
    }
  }

  /** Ends the running turn. Nothing is cleared here: the turn's ending arrives
   *  over the socket like every other ending, and clearing inFlight now would
   *  hide the last deltas still on their way. A turn that finished a moment
   *  before the click resolves false and says nothing — the click and the last
   *  delta race every time, and there is nothing wrong when the delta wins. */
  async function handleStop() {
    if (!inFlight) return;
    try {
      await stop.mutateAsync(inFlight.entryId);
    } catch (err) {
      toast.error("Could not stop the turn", {
        description: (err as Error).message,
      });
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

  const pane = (
    <>
      <PaneHeader
        title="Chat"
        action={
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={() => void handleCreate()}
            aria-label="New conversation"
          >
            <Plus className="size-4" />
          </Button>
        }
      />
      {/* A setup that was skipped or interrupted is finished from here (D-046).
          Above the conversations and below the section header, because it is
          about the account rather than about any one conversation. */}
      <SetupCard />
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
    </>
  );

  if (conversations.isError) {
    return (
      <AppShell section="chat" pane={pane} detail={false}>
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
      </AppShell>
    );
  }

  return (
    <TooltipProvider>
      <AppShell
        section="chat"
        pane={pane}
        detail={selectedId !== null}
        trayInDetail={false}
      >
        <main className="flex min-h-0 flex-1 flex-col">
          {selected ? (
            <>
              <AppHeader
                title={
                  <ConversationName title={selected.title} className="block truncate" />
                }
                back={{ onClick: () => select(null), label: "Back to conversations" }}
                subtitle={
                  /* The folder is what the agent can see, so the place it is
                     displayed is the place to change it — rather than a
                     setting somewhere you would have to know about. */
                  <button
                    type="button"
                    onClick={() => setPickingFolder(true)}
                    title={selected.folder}
                    className="flex max-w-full items-center gap-1.5 rounded text-xs leading-4 text-muted-foreground hover:text-foreground"
                  >
                    <FolderOpen className="size-3.5 shrink-0" aria-hidden />
                    <span className="truncate">{selected.folder}</span>
                    <Pencil className="size-3 shrink-0 opacity-0 transition-opacity group-hover/header:opacity-100" aria-hidden />
                  </button>
                }
              >
                <div className="hidden @xl/header:block">
                  <ChoiceToggle
                    options={VIEWS}
                    value={transcriptView}
                    onChange={setTranscriptView}
                    label="Transcript view"
                  />
                </div>
                <div className="hidden text-right font-mono text-xs text-muted-foreground @xl/header:block">
                  {/* Only claude states a dollar figure, so zero spend on a
                      conversation answered by the others means "not reported",
                      not "free". Showing it only when there is one keeps that
                      honest. */}
                  {selected.spendUsd > 0 && <div>{formatUsd(selected.spendUsd)}</div>}
                  <div>{formatTokens(selected.tokens)} tokens</div>
                </div>
              </AppHeader>

              <ProviderStrip providers={usableProviders} />

              <ScrollArea ref={scrollRef} className="min-h-0 flex-1">
                <div className="mx-auto max-w-3xl px-6 py-8">
                  {conversation.isPending ? (
                    <MessageSkeleton />
                  ) : visibleEntries.length === 0 && !inFlight ? (
                    <div className="flex flex-col items-center gap-2 py-24 text-center">
                      <p className="text-sm font-medium">Nothing said here yet</p>
                      <p className="max-w-sm text-sm text-muted-foreground">
                        Ask {pending?.to ?? selected.provider} something about{" "}
                        {selected.folder.split(/[\\/]/).pop()}. You can move this
                        conversation to another agent at any point without losing
                        it.
                      </p>
                    </div>
                  ) : (
                    <>
                      {transcriptView === "raw" ? (
                        // Every entry, the running turn's empty placeholder
                        // included: its record is worth watching before any of
                        // its answer has arrived.
                        <RawTranscript
                          conversationId={selected.id}
                          entries={selected.entries}
                          runningId={streamingId}
                        />
                      ) : (
                        <MessageList
                          entries={visibleEntries}
                          streamingId={streamingId}
                        />
                      )}
                      {inFlight && (
                        <div className="mt-8">
                          <WorkingIndicator
                            provider={inFlight.provider}
                            model={inFlight.model}
                            startedAt={inFlight.startedAt}
                            streams={inFlightProvider?.streams ?? false}
                          />
                        </div>
                      )}
                    </>
                  )}
                </div>
              </ScrollArea>

              <FolderPicker
                open={pickingFolder}
                onOpenChange={setPickingFolder}
                current={selected.folder}
                hasMessages={selected.entries.length > 0}
                onChoose={(folder) =>
                  setFolder.mutate({ id: selected.id, patch: { folder } })
                }
              />

              <Composer
                providers={usableProviders}
                activeProvider={activeProvider}
                activeModel={activeModel}
                pending={pending}
                disabled={!canReach || daemonTooOld || inFlight !== null}
                disabledTone={noComputerYet || noComputerInWorkspace ? "neutral" : "warning"}
                disabledReason={
                  noComputerYet
                    ? "You have not connected a computer yet, so there is nothing to run agents on. Connect one in Machines. Everything already said stays readable."
                    : noComputerInWorkspace
                      ? // A choice somebody made, not a fault, so neutral like
                        // the one above (B-46), with the place to change it.
                        "This workspace has no computer, so nothing can run in it. Give it one in Settings → Workspaces. Everything already said stays readable."
                    : !canReach
                      ? "Your machine is unreachable, so nothing new can be sent. Everything already said stays readable."
                      : daemonTooOld
                        ? "Your computer's sparstrowgen is too old for this app, so nothing new can be sent. Update it in Settings → Updates. Everything already said stays readable."
                        : undefined
                }
                value={draft}
                onChange={(v) => selectedId && setDraft(selectedId, v)}
                onSelect={(p, m) => void handleSelectProvider(p, m)}
                onCancelSwitch={() => setPending(null)}
                onSend={() => void handleSend()}
                running={inFlight !== null}
                onStop={() => void handleStop()}
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
      </AppShell>
    </TooltipProvider>
  );
}
