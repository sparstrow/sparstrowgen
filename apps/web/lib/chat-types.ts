/* Shapes the chat surface renders. These mirror what the daemon can actually
   report — see docs/Capabilities.md. Anything optional here is optional because
   some provider genuinely does not emit it, not for convenience. */

export type ProviderId = "claude" | "codex" | "agy" | "gemini";

/** Three states, not two. The distinction is whether waiting is a plan.
 *  See docs/Decisions.md D-011. */
export type Availability = "available" | "waitable" | "blocked";

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
  models: string[];
  /** null when the provider cannot be used at all. */
  model: string | null;
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
  model: string;
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
  toModel: string;
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
  model: string;
  spendUsd: number;
  tokens: number;
  entries: Entry[];
  /** Which providers have already seen how much of this conversation, so a
   *  switch back only replays the gap. Keyed by provider id → entries seen. */
  seenBy: Partial<Record<ProviderId, number>>;
};

/** What a switch would cost, computed but not yet incurred. */
export type PendingSwitch = {
  to: ProviderId;
  toModel: string;
  messagesToReplay: number;
  estimatedTokens: number;
};
