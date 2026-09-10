/* Placeholder data for the chat surface.
   A shipped route must never import this — grep for ".mock" before calling the
   feature done (AGENTS.md §4). Deliberately untidy: a very long title, an
   untitled conversation, an empty one, uneven message lengths, a failed turn,
   and two archived.

   MODEL LISTS ARE REAL, captured 2026-09-10 — see docs/Capabilities.md for how
   each was obtained and which of them the CLI can actually enumerate. Do not
   add a model here from memory; that is the mistake this round corrected. */

import type { Conversation, Model, Provider } from "./chat-types";

/* Resolved by running each documented alias and reading the id back out of the
   `system.init` line — `claude -p --model opus` reports `claude-opus-4-6`. Not
   from memory: an earlier version of this list said "Opus 5" / "Sonnet 5",
   which this machine's CLI does not offer. */
const claudeModels: Model[] = [
  { id: "claude-opus-4-6", label: "Opus 4.6" },
  { id: "claude-sonnet-4-6", label: "Sonnet 4.6" },
  { id: "claude-haiku-4-5-20251001", label: "Haiku 4.5" },
];

const codexModels: Model[] = [
  { id: "gpt-5.6-sol", label: "GPT-5.6 Sol" },
  { id: "gpt-5.6-luna", label: "GPT-5.6 Luna" },
  { id: "gpt-5.6-terra", label: "GPT-5.6 Terra" },
];

/* Verbatim from `agy models`. Effort is part of the id, not a separate axis —
   which is why there is no effort control in the UI: picking "Gemini 3.1 Pro
   (High)" IS picking the effort. */
const agyModels: Model[] = [
  { id: "gemini-3.8-flash-high", label: "Gemini 3.8 Flash (High)" },
  { id: "gemini-3.8-flash-medium", label: "Gemini 3.8 Flash (Medium)" },
  { id: "gemini-3.8-flash-low", label: "Gemini 3.8 Flash (Low)" },
  { id: "gemini-3.7-flash-high", label: "Gemini 3.7 Flash (High)" },
  { id: "gemini-3.7-flash-medium", label: "Gemini 3.7 Flash (Medium)" },
  { id: "gemini-3.7-flash-low", label: "Gemini 3.7 Flash (Low)" },
  { id: "gemini-3.6-flash-high", label: "Gemini 3.6 Flash (High)" },
  { id: "gemini-3.6-flash-medium", label: "Gemini 3.6 Flash (Medium)" },
  { id: "gemini-3.6-flash-low", label: "Gemini 3.6 Flash (Low)" },
  { id: "gemini-3.1-pro-high", label: "Gemini 3.1 Pro (High)" },
  { id: "gemini-3.1-pro-low", label: "Gemini 3.1 Pro (Low)" },
  { id: "claude-sonnet-4-6", label: "Claude Sonnet 4.6 (Thinking)" },
  { id: "claude-opus-4-6-thinking", label: "Claude Opus 4.6 (Thinking)" },
  { id: "gpt-oss-120b-medium", label: "GPT-OSS 120B (Medium)" },
];

export const mockProviders: Provider[] = [
  {
    id: "claude",
    label: "claude",
    models: claudeModels,
    model: claudeModels[1],
    availability: "available",
    headroom: { pct: 62, resetsIn: "4h 12m", window: "five_hour" },
    reportsUsd: true,
    streams: true,
    routes: false,
  },
  {
    id: "codex",
    label: "codex",
    models: codexModels,
    model: codexModels[0],
    availability: "available",
    headroom: null,
    reportsUsd: false,
    streams: false,
    routes: false,
  },
  {
    id: "agy",
    label: "agy",
    models: agyModels,
    model: agyModels[9],
    availability: "available",
    headroom: null,
    reportsUsd: false,
    streams: true,
    routes: true,
  },
];

const sonnet = claudeModels[1];
const opus = claudeModels[0];
const sol = codexModels[0];
const geminiPro = agyModels[9];

export const mockConversations: Conversation[] = [
  {
    id: "c1",
    title: "daemon reconnect backoff",
    folder: "D:\\sparstrowgen",
    updated: "2m",
    provider: "codex",
    model: sol,
    spendUsd: 0.14,
    tokens: 41700,
    seenBy: { claude: 4, agy: 4, codex: 6 },
    archived: false,
    entries: [
      {
        id: "m1",
        role: "user",
        at: "09:12",
        text: "The daemon drops its socket when the laptop sleeps, then hammers the server on wake. Can you add reconnect backoff?",
      },
      {
        id: "m2",
        role: "agent",
        at: "09:12",
        provider: "claude",
        model: sonnet,
        usage: { tokens: 12400, usd: 0.03 },
        text: "The reconnect loop should treat a dropped socket as a transient failure rather than a new session. I'd keep the attempt counter beside the connection state, reset it only once the hub confirms a stable subscription, and add bounded exponential backoff with full jitter so a fleet of sleeping clients doesn't all wake into the same millisecond.",
        code: {
          lang: "go",
          body: "for attempt := 0; ; attempt++ {\n    delay := backoff(attempt)\n    if !sleepCtx(ctx, delay) {\n        return ctx.Err()\n    }\n    if err := c.dial(ctx); err == nil {\n        return nil\n    }\n}",
        },
      },
      {
        id: "m3",
        role: "user",
        at: "09:16",
        text: "Agreed. Keep the transcript readable while it's recovering — I don't want the whole window greying out.",
      },
      {
        id: "m4",
        role: "agent",
        at: "09:17",
        provider: "agy",
        model: geminiPro,
        usage: { tokens: 8100 },
        text: "Then the reconnect state belongs to the transport, not the view. Show a quiet reconnecting line in the status strip, keep existing messages fully readable, and disable only the controls that genuinely need a live socket — the composer and the provider switch. If recovery exceeds the retry window, surface one clear error with a manual retry rather than stacking a notice per attempt.",
      },
      {
        id: "r1",
        role: "replay",
        at: "09:20",
        to: "codex",
        toModel: sol,
        messagesReplayed: 4,
        tokens: 18200,
      },
      {
        id: "m5",
        role: "agent",
        at: "09:21",
        provider: "codex",
        model: sol,
        usage: { tokens: 12400 },
        text: "I traced the retry path through the daemon and kept cancellation intact end to end. The jitter source is injected rather than read from the global rand, which is what makes the timing test deterministic. Registration and broadcast are single-owner operations inside the hub goroutine, so shutdown has exactly one ordering: stop accepting clients, close subscriptions, drain pending notifications, then close sockets.",
      },
    ],
  },
  {
    id: "c2",
    title: "invoice sync — EDI 810 segment mapping against the vendor's spec revision",
    folder: "D:\\aim-clinic",
    updated: "1h",
    provider: "codex",
    model: sol,
    spendUsd: 0,
    tokens: 96400,
    seenBy: { codex: 22 },
    archived: false,
    entries: [
      {
        id: "e1",
        role: "user",
        at: "08:04",
        text: "The 810 they send has REF*IV in the header loop, not the detail loop. Does our mapper handle that?",
      },
      {
        id: "e2",
        role: "agent",
        at: "08:05",
        provider: "codex",
        model: sol,
        usage: { tokens: 9600 },
        text: "It does not. The mapper assumes REF segments are detail-scoped and will silently drop a header-level invoice reference, which is why three of last week's invoices came through without a number. The fix is to resolve REF at both scopes and let the detail value win when both are present.",
      },
    ],
  },
  {
    id: "c3",
    title: "clinic booking flow",
    folder: "D:\\aim-clinic",
    updated: "yesterday",
    provider: "claude",
    model: opus,
    spendUsd: 1.82,
    tokens: 210300,
    seenBy: { claude: 2 },
    archived: false,
    entries: [
      {
        id: "b1",
        role: "user",
        at: "16:41",
        text: "Walk me through what happens when two parents book the same slot within a second of each other.",
      },
      {
        id: "b2",
        role: "agent",
        at: "16:42",
        provider: "claude",
        model: opus,
        usage: { tokens: 15200, usd: 0.41 },
        text: "Right now, both succeed. The availability check and the insert are separate statements with no constraint behind them, so two requests can both read the slot as free before either writes. The second parent gets a confirmation for a slot that is already taken, and nobody finds out until the clinic opens the day view.",
        failure: "Connection to the daemon was lost before this turn finished.",
      },
    ],
  },
  {
    id: "c4",
    title: "postgres migration — pgvector index tuning for the embeddings table",
    folder: "D:\\sparstrowgen",
    updated: "3d",
    provider: "agy",
    model: geminiPro,
    spendUsd: 0,
    tokens: 54100,
    seenBy: { agy: 1 },
    archived: false,
    entries: [
      {
        id: "p1",
        role: "user",
        at: "11:02",
        text: "Is an HNSW index worth it at 40k rows, or is a flat scan still fine?",
      },
    ],
  },
  {
    id: "c5",
    title: "Untitled conversation",
    folder: "D:\\sparstrowgen",
    updated: "4d",
    provider: "claude",
    model: sonnet,
    spendUsd: 0,
    tokens: 0,
    seenBy: {},
    archived: false,
    entries: [],
  },
  {
    id: "c6",
    title: "Untitled conversation",
    folder: "D:\\aim-clinic",
    updated: "2w",
    provider: "claude",
    model: sonnet,
    spendUsd: 0.44,
    tokens: 38200,
    seenBy: { claude: 2 },
    archived: true,
    entries: [
      {
        id: "a1",
        role: "user",
        at: "10:20",
        text: "Why does the Clockify export round every entry up to the next quarter hour?",
      },
      {
        id: "a2",
        role: "agent",
        at: "10:21",
        provider: "claude",
        model: sonnet,
        usage: { tokens: 11800, usd: 0.09 },
        text: "Because the rounding is applied per entry on the way out rather than once on the invoice total. Six eight-minute calls become six fifteen-minute lines, which is where the extra 42 minutes on last month's statement came from. Round the summed duration, not each row.",
      },
    ],
  },
  {
    id: "c7",
    title: "NAV item ledger — costing adjustment rerun",
    folder: "D:\\aim-clinic",
    updated: "1mo",
    provider: "agy",
    model: agyModels[0],
    spendUsd: 0,
    tokens: 12600,
    seenBy: { agy: 1 },
    archived: true,
    entries: [
      {
        id: "n1",
        role: "user",
        at: "14:55",
        text: "The adjust cost batch job ran for nine hours and still left value entries open. Where do I even start?",
      },
    ],
  },
];

/** Canned reply used when the prototype simulates a turn. */
export const mockReply = {
  text: "Bounded backoff is in place, and the attempt counter now resets only after a healthy subscription rather than after a successful dial — those are different moments, and the second one is what was causing the reconnect storm. I added a fake clock to the timing test so the retry schedule is asserted exactly instead of slept through, and the cancellation path returns the context error rather than swallowing it.",
  code: {
    lang: "go",
    body: "func backoff(attempt int) time.Duration {\n    d := base << attempt\n    if d > maxDelay {\n        d = maxDelay\n    }\n    return time.Duration(rand.Int63n(int64(d)))\n}",
  },
};
