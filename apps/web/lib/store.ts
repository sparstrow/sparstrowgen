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

  /** The turn in flight in each conversation, keyed by conversation id, so that
   *  conversation's composer can lock and its working indicator can tick. A turn
   *  running in one conversation locks only that one (docs/Bugs.md B-29).
   *  startedAt is stored rather than an elapsed count, so the display is derived
   *  from the clock instead of accumulated by a timer that has to be reset. */
  inFlight: Record<string, InFlightTurn>;
  setInFlight: (conversationId: string, turn: InFlightTurn | null) => void;

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

export type InFlightTurn = {
  entryId: string;
  provider: ProviderId;
  model: Model;
  startedAt: number;
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

  inFlight: {},
  setInFlight: (conversationId, turn) =>
    set((s) => {
      const next = { ...s.inFlight };
      if (turn) next[conversationId] = turn;
      else delete next[conversationId];
      return { inFlight: next };
    }),

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
/* -------------------------------------------------------------------------
   The shell.

   Whether the navigation rail is held open. It is a choice about one window,
   not about the account (docs/Decisions.md D-043), so it is remembered in this
   browser and never sent anywhere: no column, no endpoint. Reading
   localStorage is wrapped because it throws outright in a browser with site
   data blocked, and a pin nobody can save is not worth a blank screen. */

const RAIL_PIN_KEY = "sparstrowgen.rail-pinned";

function readPinned(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(RAIL_PIN_KEY) === "1";
  } catch {
    return false;
  }
}

type ShellView = {
  railPinned: boolean;
  toggleRailPin: () => void;
  /** Called once on mount, because the server renders with the rail closed and
   *  only the browser can know better. */
  loadRailPin: () => void;
};

export const useShellView = create<ShellView>((set, get) => ({
  railPinned: false,
  toggleRailPin: () => {
    const next = !get().railPinned;
    set({ railPinned: next });
    try {
      window.localStorage.setItem(RAIL_PIN_KEY, next ? "1" : "0");
    } catch {
      // Blocked site data: the pin still works for this visit.
    }
  },
  loadRailPin: () => set({ railPinned: readPinned() }),
}));

/* -------------------------------------------------------------------------
   Skipping first-run setup.

   Per browser, like the rail pin, and for a sharper reason than convenience:
   the only step that exists is connecting a computer, and the daemon runs on
   the computer this browser is running on. Opening the app on a phone and
   being asked about connecting *that* device is a genuinely different
   question, so answering it once here should not answer it everywhere
   (docs/Decisions.md D-046).

   Profile and Workspace are account facts rather than browser ones, which this
   comment used to say would move the flag to the account. It has not, and here
   is why it does not need to: skipping means "do not stop me on the way in",
   and that is exactly a per-browser wish. What changed instead is that the one
   step the app cannot run without — having a workspace at all — is not
   skippable, and the redirect in app/page.tsx ignores this flag in that one
   case. A skip can defer a name and a picture; it cannot defer the thing every
   conversation needs to be in. */

const SETUP_SKIPPED_KEY = "sparstrowgen.setup-skipped";

function readSkipped(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(SETUP_SKIPPED_KEY) === "1";
  } catch {
    return false;
  }
}

type SetupView = {
  /** Whether setup has been skipped in this browser. Starts true on the server
   *  and before the flag is read, so nobody is briefly redirected into a wizard
   *  they already dismissed. */
  skipped: boolean;
  /** True once the flag has actually been read, so the redirect can wait for a
   *  real answer instead of acting on the default. */
  loaded: boolean;
  skip: () => void;
  /** Finishing setup, or choosing to resume it, clears the skip. */
  unskip: () => void;
  loadSkipped: () => void;
  /** The × on the resume card. Deliberately NOT persisted: whether a hidden
   *  card should stay hidden across visits is an open question for the owner
   *  (the prototype handoff's OQ7), and a flag written to disk would answer it
   *  silently. For this visit it goes away; on the next it is back. */
  cardHidden: boolean;
  hideCard: () => void;
};

export const useSetupView = create<SetupView>((set) => ({
  skipped: true,
  loaded: false,
  cardHidden: false,
  hideCard: () => set({ cardHidden: true }),
  skip: () => {
    set({ skipped: true });
    try {
      window.localStorage.setItem(SETUP_SKIPPED_KEY, "1");
    } catch {
      // Blocked site data: setup stays skipped for this visit only.
    }
  },
  unskip: () => {
    set({ skipped: false });
    try {
      window.localStorage.removeItem(SETUP_SKIPPED_KEY);
    } catch {
      // As above.
    }
  },
  loadSkipped: () => set({ skipped: readSkipped(), loaded: true }),
}));

/* -------------------------------------------------------------------------
   Which workspace is being looked at.

   View state, and deliberately so (docs/Decisions.md D-050). Two tabs open on
   two workspaces is the normal way to use this, and a "current workspace" kept
   on the account would make that the broken case — switching in one tab would
   move the other.

   Remembered in this browser so coming back lands where you left off, and read
   in an effect rather than in the store's initial state so the server and
   client render the same markup. The id is only ever a hint: the workspace
   list is the authority, and `resolveWorkspace` below drops one this account
   can no longer reach. */

const WORKSPACE_KEY = "sparstrowgen.workspace";

function readWorkspace(): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage.getItem(WORKSPACE_KEY);
  } catch {
    return null;
  }
}

type WorkspaceView = {
  /** The remembered id, before it has been checked against the real list. */
  workspaceId: string | null;
  /** True once the browser's remembered value has been read. Until then there
   *  is no answer, only a default, and nothing should act on it. */
  loaded: boolean;
  setWorkspace: (id: string) => void;
  loadWorkspace: () => void;
};

export const useWorkspaceView = create<WorkspaceView>((set) => ({
  workspaceId: null,
  loaded: false,
  setWorkspace: (workspaceId) => {
    set({ workspaceId });
    try {
      window.localStorage.setItem(WORKSPACE_KEY, workspaceId);
    } catch {
      // Blocked site data: the choice holds for this visit.
    }
  },
  loadWorkspace: () => set({ workspaceId: readWorkspace(), loaded: true }),
}));

/* ---------------------------------------------------------------------- */

export const selectSelectedId = (s: ChatView) => s.selectedId;
export const selectPending = (s: ChatView) => s.pending;
export const selectInFlight = (s: ChatView) => s.inFlight;
export const selectSearch = (s: ChatView) => s.search;
export const selectTranscriptView = (s: ChatView) => s.transcriptView;
