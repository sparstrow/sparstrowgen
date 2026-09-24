/* Shapes the chat surface renders. These mirror what the daemon can actually
   report — see docs/Capabilities.md. Anything optional here is optional because
   some provider genuinely does not emit it, not for convenience.

   The SERVER owns these shapes: server/internal/protocol/protocol.go carries
   json tags that must match this file field for field. Protobuf would have
   generated both from one source and was deferred (docs/Decisions.md D-017), so
   until it lands, changing one side without the other is a runtime bug rather
   than a compile error. Change them in the same commit. */

/** gemini is deliberately absent. It is installed on the machine but has no
 *  account behind it and the owner has ruled it out, so there is no adapter and
 *  no reason to carry it through the type system. See docs/Decisions.md D-014. */
export type ProviderId = "claude" | "codex" | "agy";

/** Three states, not two. The distinction is whether waiting is a plan.
 *  See docs/Decisions.md D-011. */
export type Availability = "available" | "waitable" | "blocked";

/** A model always has both. `agy models` returns `gemini-3.1-pro-high` next to
 *  "Gemini 3.1 Pro (High)" — the id is what gets passed to the CLI, the label is
 *  the only thing worth showing. Never derive one from the other. */
export type Model = {
  id: string;
  label: string;
};

/** What claude's rate_limit_event actually carries.
 *
 *  There is no percentage in it. The first design showed "62%" behind a
 *  capacity bar and that figure was invented — the real payload is a status, a
 *  reset timestamp and a window name. So we can say honestly *when the window
 *  resets*, and never *how much is left*. See docs/Decisions.md D-018. */
export type Headroom = {
  /** "allowed" is the only value ever observed. */
  status: string;
  /** Unix seconds. Counted down on the client so it stays live. */
  resetsAt: number;
  /** e.g. "five_hour". */
  window: string;
};

export type Provider = {
  id: ProviderId;
  label: string;
  models: Model[];
  /** null when the provider cannot be used at all. */
  model: Model | null;
  availability: Availability;
  /** Present only when availability is not "available". */
  unavailableReason?: string;
  /** null means the provider reports no limit signal — NOT that it has none left.
   *  Only claude emits rate_limit_event. See KnownGaps G-4. */
  headroom: Headroom | null;
  /** Whether this provider reports real currency. Only claude does. */
  reportsUsd: boolean;
  /** Whether it emits a usage window at all. Distinct from `headroom` being
   *  null: a provider that reports limits but has not been used yet is "not
   *  known yet", which is a different claim from "reports nothing". */
  reportsLimits: boolean;
  /** Whether text arrives incrementally. codex VERIFIED false — one whole
   *  message per turn. See KnownGaps G-5 for claude. */
  streams: boolean;
  /** True when the provider is a router in front of several vendors' models
   *  rather than one vendor's own CLI. agy serves Gemini, Claude and GPT models,
   *  so "provider" here means "the CLI we drive", never "who made the model". */
  routes: boolean;
};

export type Usage = {
  tokens: number;
  /** Only ever set for providers with reportsUsd. */
  usd?: number;
};

export type Machine = {
  id: string;
  name: string;
  approved: boolean;
  createdAt: string;
  lastSeenAt?: string;
  online: boolean;
  providers: Provider[];
  /** The daemon version it reports, or last reported while offline. Absent for
   *  a computer that has never said. */
  version?: string;
  /** Too old for the hosted app: it is sent no work until it is updated. */
  tooOld: boolean;
  /** Whether its version can update itself. Only known while it is online. */
  selfUpdates: boolean;
  automaticUpdates: boolean;
  update: UpdateStatus;
};

/** Where an update stands for one computer (spec US3). */
export type UpdateStatus =
  | { kind: "unchecked" }
  | { kind: "current" }
  | { kind: "available"; version: string }
  | { kind: "waiting"; version: string; activeTasks: number }
  | { kind: "updating"; version: string }
  | { kind: "failed"; message: string };

export type ComputerUpdates = {
  machineId: string;
  name: string;
  online: boolean;
  /** As last reported by the computer. */
  version: string;
  /** The hosted app no longer works with this version; it cannot update itself. */
  tooOld: boolean;
  automatic: boolean;
  status: UpdateStatus;
};

/** Who a person is, as opposed to how they sign in. Per account, like
 *  appearance, so it follows them to every browser. */
export type Profile = {
  /** Empty means not set — fall back to the email address. There is
   *  deliberately no second way to be absent. */
  displayName: string;
  bio: string;
  /** Absent when there is no picture. Doubles as the picture's cache key, so
   *  replacing one is a new URL. */
  avatarUpdatedAt?: string;
};

export type Pairing = {
  id: string;
  status: "pending" | "claimed" | "approved" | "rejected";
  machineId?: string;
  expiresAt: string;
  /** The computer that claimed this request. Absent until one does, and never
   *  in the machines list — that is approved computers only. It is what the
   *  approval prompt names, so the person can see what they are approving. */
  machineName?: string;
};

export type UserMessage = {
  id: string;
  role: "user";
  at: string;
  text: string;
};

export type AgentMessage = {
  id: string;
  role: "agent";
  at: string;
  provider: ProviderId;
  /** The whole model, not its id. A transcript has to stay readable after a
   *  provider drops a model from its list — at that point nothing can resolve
   *  the label any more, so it is stored at the time the turn ran. */
  model: Model;
  text: string;
  usage?: Usage;
  /** Set when the turn ended badly. Text may still hold a partial answer. */
  failure?: string;
  /** The owner ended this turn rather than the agent finishing it. Kept apart
   *  from `failure` because they read differently: one is something going
   *  wrong, the other is a decision. Both can be set when a CLI complained on
   *  its way out — the surface leads with the stop, since the complaint is a
   *  consequence of it. */
  stopped?: boolean;
};

/** A provider switch that has actually been paid for, recorded in the transcript
 *  at the point the replay happened — never at the point the provider was
 *  selected. Selecting is free; catching up is not. */
export type ReplayMarker = {
  id: string;
  role: "replay";
  at: string;
  to: ProviderId;
  toModel: Model;
  messagesReplayed: number;
  tokens: number;
};

export type Entry = UserMessage | AgentMessage | ReplayMarker;

/** A separate area of work inside one account (docs/Decisions.md D-050). One
 *  for personal things, one for work; conversations live in one and never
 *  appear in another. `role` is this account's role in it, which is what
 *  decides whether renaming is offered. */
export type Workspace = {
  id: string;
  name: string;
  role: "owner" | "member";
};

/** One of the account's computers, with whether a particular workspace may use
 *  it (migration 00018). Every computer is listed either way, because the
 *  surface is a set of checkboxes over all of them. */
export type AssignedMachine = Machine & {
  assigned: boolean;
  /** Whether it is connected right now, so an assignment can be made without
   *  guessing which of two names is the laptop that is awake. */
  online: boolean;
};

/** The same assignment from the computer's end: which workspaces offer it. */
export type AssignedWorkspace = Workspace & { assigned: boolean };

export type Conversation = {
  id: string;
  /** Which workspace it lives in. Carried on every conversation because live
   *  events are addressed to an ACCOUNT rather than to a workspace: without it
   *  a tab looking at Personal would patch a Work conversation into its list. */
  workspaceId: string;
  /** Empty until it has a name — from the first thing said in it, or from the
   *  owner typing one. The surface shows a placeholder for that, which
   *  describes a conversation with no name rather than pretending to be one. */
  title: string;
  /** The directory the agents run in. A conversation is always about somewhere. */
  folder: string;
  updated: string;
  provider: ProviderId;
  model: Model;
  spendUsd: number;
  tokens: number;
  entries: Entry[];
  /** Which providers have already seen how much of this conversation, so a
   *  switch back only replays the gap. Keyed by provider id → entries seen. */
  seenBy: Partial<Record<ProviderId, number>>;
  /** Set only on a search result whose match was in the message text: the
   *  matching line, so a hit in a long transcript is explicable. A title match
   *  leaves this empty — the reason for that hit is already on screen. */
  excerpt?: string;
  /** Out of the list but fully intact. Unlike deleting, this is reversible,
   *  which is why it is offered at the point of deletion. */
  archived: boolean;
};

/** What a switch would cost, computed but not yet incurred. */
export type PendingSwitch = {
  to: ProviderId;
  toModel: Model;
  messagesToReplay: number;
  estimatedTokens: number;
};


/* ── A turn's exchange ────────────────────────────────────────────────────────
   Everything that crossed between the daemon and one CLI in one turn
   (docs/specs/2026-09-23-raw-exchange.md). Mirrors protocol.Exchange. */

/** What the CLI was handed. The environment is never here: it carries the
 *  sign-in token. */
export type ExchangeSent = {
  /** The resolved executable, or only the provider's name when it never started. */
  program: string;
  args: string[];
  cwd: string;
  /** Absent when the turn started a new session. */
  resumeSessionId?: string;
  /** The prompt as the agent reads it, catch-up included. */
  prompt: string;
  /** The exact bytes written to stdin. Empty when it never started. */
  stdin: string;
  /** False when the daemon refused the turn before starting the CLI. */
  launched: boolean;
};

export type ExchangeLine = {
  /** Order of arrival, from 1. A gap means a batch went missing. */
  seq: number;
  /** Milliseconds since the CLI started. */
  atMs: number;
  stream: "stdout" | "stderr";
  text: string;
};

/** A CLI's own account of its tokens. `input` and `output` are the whole of
 *  each; the optional fields are parts of them, absent when not broken out. */
export type ExchangeUsage = {
  input: number;
  fromCache?: number;
  toCache?: number;
  output: number;
  reasoning?: number;
  total: number;
};

/** What the CLI said about itself, read from its lines on the server. A field it
 *  did not report is absent, never guessed. */
export type ExchangeReport = {
  cliVersion?: string;
  model?: string;
  permissionMode?: string;
  cwd?: string;
  sessionId?: string;
  tools?: string[];
  skills?: string[];
  agents?: string[];
  mcpServers?: string[];
  usage?: ExchangeUsage;
};

/** How big one turn's record is, without its contents. */
export type ExchangeSummary = {
  entryId: string;
  launched: boolean;
  lines: number;
  /** UTF-8 bytes of the lines kept. */
  bytes: number;
  /** When the last line arrived, in ms since the CLI started. */
  lastAtMs: number;
  dropped?: { lines: number; bytes: number };
};

export type Exchange = {
  entryId: string;
  /** False for a turn with no record: from before recording, from a daemon that
   *  does not record, or one that never reached a computer. */
  recorded: boolean;
  sent?: ExchangeSent;
  lines: ExchangeLine[];
  /** What was not kept once the turn's record was full. */
  dropped?: { lines: number; bytes: number };
  report?: ExchangeReport;
};


/* ── Directories ──────────────────────────────────────────────────────────────
   Only the daemon can see the owner's filesystem, so browsing goes through it
   (docs/Decisions.md D-020). These mirror protocol.DirListing. */

/** Why a directory cannot be used, as a value rather than a sentence — so the
 *  wording lives here, in the surface that shows it, and a client never parses
 *  prose to tell "you pasted a file" from "that folder is read-only". */
export type DirReason =
  | ""
  | "not_absolute"
  | "not_found"
  | "not_a_directory"
  | "not_readable";

export type DirEntry = {
  name: string;
  path: string;
};

export type DirListing = {
  /** As the daemon resolved it: absolute, with ".." and symlinks collapsed. */
  path: string;
  /** Empty at a root, so the picker knows not to offer "up". */
  parent: string;
  /** Subdirectories only. A conversation runs in a directory, so listing files
   *  would offer a choice that cannot be made. */
  entries: DirEntry[];
  reason: DirReason;
  /** Advisory. Shown so a folder outside any repo is noticeable before the
   *  conversation starts, never to prevent the choice. */
  isGitRepo: boolean;
};
