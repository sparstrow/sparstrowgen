import { create } from "zustand";
import type { Model, PendingSwitch, ProviderId } from "./chat-types";

/* View state only. Nothing the server owns may live here.
 *
 * AGENTS.md §3: TanStack Query owns server state, Zustand owns view state, and
 * realtime events invalidate or patch the Query cache and must NEVER be
 * mirrored into a store. Mirroring is what creates two sources of truth for one
 * fact, and then a bug where they disagree.
 *
 * Everything below is genuinely local: which conversation is open, what has
 * been typed, what switch is being contemplated, what is in the search box.
 * None of it survives a refresh, and none of it should. */

export type TranscriptView = "rendered" | "raw";

type ChatView = {
  /** Which conversation is open. An id, not the conversation — the object
   *  belongs to the Query cache. */
  selectedId: string | null;
  select: (id: string | null) => void;

  /** Unsent text, per conversation, so switching away and back does not throw
   *  away what was typed. */
  drafts: Record<string, string>;
  draftFor: (id: string | null) => string;
  setDraft: (id: string, text: string) => void;
  clearDraft: (id: string) => void;

  /** A switch that has been chosen but not paid for. This deliberately never
   *  persists: selecting a provider costs nothing, and the catch-up is charged
   *  on the next message. A pending switch that survived a reload would imply
   *  a commitment that was never made. */
  pending: PendingSwitch | null;
  setPending: (p: PendingSwitch | null) => void;

  /** Set while a turn is in flight, so the composer can lock and the working
   *  indicator can tick. startedAt is stored rather than an elapsed count, so
   *  the display is derived from the clock instead of accumulated by a timer
   *  that has to be reset. */
  inFlight: {
    entryId: string;
    provider: ProviderId;
    model: Model;
    startedAt: number;
  } | null;
  setInFlight: (t: ChatView["inFlight"]) => void;

  search: string;
  setSearch: (q: string) => void;

  /** Rendered markdown, or the stored text verbatim. A view mode rather than a
   *  per-message toggle: it is used to check what the renderer is dropping,
   *  and that question is asked of a conversation, not of one reply. */
  transcriptView: TranscriptView;
  setTranscriptView: (v: TranscriptView) => void;

  /** Archived conversations are hidden behind a disclosure unless searching. */
  showArchived: boolean;
  toggleArchived: () => void;
};

export const useChatView = create<ChatView>((set, get) => ({
  selectedId: null,
  // Selecting a different conversation abandons any switch being considered:
  // the quote was about the old transcript and means nothing against this one.
  select: (id) => set({ selectedId: id, pending: null }),

  drafts: {},
  draftFor: (id) => (id ? (get().drafts[id] ?? "") : ""),
  setDraft: (id, text) => set((s) => ({ drafts: { ...s.drafts, [id]: text } })),
  clearDraft: (id) =>
    set((s) => {
      const next = { ...s.drafts };
      delete next[id];
      return { drafts: next };
    }),

  pending: null,
  setPending: (pending) => set({ pending }),

  inFlight: null,
  setInFlight: (inFlight) => set({ inFlight }),

  search: "",
  setSearch: (search) => set({ search }),

  transcriptView: "rendered",
  setTranscriptView: (transcriptView) => set({ transcriptView }),

  showArchived: false,
  toggleArchived: () => set((s) => ({ showArchived: !s.showArchived })),
}));

/* Selectors return primitives or stable references so a component only
   re-renders when the value it actually reads changes. Returning a fresh object
   from a selector re-renders on every store write, which is the standard way
   this pattern goes wrong. */
export const selectSelectedId = (s: ChatView) => s.selectedId;
export const selectPending = (s: ChatView) => s.pending;
export const selectInFlight = (s: ChatView) => s.inFlight;
export const selectSearch = (s: ChatView) => s.search;
export const selectTranscriptView = (s: ChatView) => s.transcriptView;
