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

export type Conversation = {
  id: string;
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
