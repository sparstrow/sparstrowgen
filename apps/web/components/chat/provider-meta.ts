import type { ProviderId } from "@/lib/chat-types";

/* Tailwind scans source for complete class names, so provider classes are
   written out rather than composed at runtime. A template literal like
   `text-provider-${id}` produces no CSS. */
export const providerClasses: Record<
  ProviderId,
  { dot: string; text: string; rule: string }
> = {
  claude: {
    dot: "bg-provider-claude",
    text: "text-provider-claude",
    rule: "bg-provider-claude/40",
  },
  codex: {
    dot: "bg-provider-codex",
    text: "text-provider-codex",
    rule: "bg-provider-codex/40",
  },
  agy: {
    dot: "bg-provider-agy",
    text: "text-provider-agy",
    rule: "bg-provider-agy/40",
  },
};

/** The classes for a provider we have never heard of.
 *
 *  Neutral rather than alarming: an unknown provider is not an error, it is a
 *  server that knows about something this build does not. Colouring it as a
 *  fault would be a lie about what happened.
 */
const unknownProvider = {
  dot: "bg-muted-foreground",
  text: "text-muted-foreground",
  rule: "bg-muted-foreground/40",
};

/** Provider classes that always return something.
 *
 *  Indexing the map directly returns `undefined` for anything not in it, and
 *  the very next line reads `.text` off it — so ONE row carrying a provider
 *  this build does not know white-screens the entire surface it appears on.
 *  That is not hypothetical: a conversation created while the daemon was
 *  offline has no provider at all, and seven of them took the sidebar down.
 *
 *  It is also the case AGENTS.md §3 names directly — an installed daemon will
 *  one day be older than the server, so a value arriving from outside is
 *  defaulted deliberately rather than trusted to be in the enum.
 */
export function providerStyle(provider: ProviderId | string | undefined) {
  return providerClasses[provider as ProviderId] ?? unknownProvider;
}

export function formatTokens(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

export function formatUsd(n: number): string {
  return `$${n.toFixed(2)}`;
}
