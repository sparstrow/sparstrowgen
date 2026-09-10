/* Shapes the chat surface renders. These mirror what the daemon can actually
   report — see docs/Capabilities.md. Anything optional here is optional because
   some provider genuinely does not emit it, not for convenience. */

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

export type Headroom = {
  /** Percent of the current window remaining. */
  pct: number;
  /** Human label for when the window resets, e.g. "4h 12m". */
  resetsIn: string;
  /** Which window this is — claude reports "five_hour". */
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
  code?: { lang: string; body: string };
  /** Set when the turn ended badly. Text may still hold a partial answer. */
  failure?: string;
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

/** Where a search term was found. A title-only match is not enough to find the
 *  conversations that most need finding — several are called "Untitled
 *  conversation". */
export type SearchHit = {
  /** The matching line of message text, when the match was not in the title. */
  excerpt?: string;
  inTitle: boolean;
};
