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
  gemini: {
    dot: "bg-provider-gemini",
    text: "text-provider-gemini",
    rule: "bg-provider-gemini/40",
  },
};

/** Headroom colour steps. A provider that reports nothing gets "unknown",
 *  which is not on the ok→out ramp on purpose — silence must never read as
 *  healthy. */
export function capacityClass(pct: number | null): string {
  if (pct === null) return "bg-capacity-unknown";
  if (pct <= 10) return "bg-capacity-out";
  if (pct <= 30) return "bg-capacity-low";
  return "bg-capacity-ok";
}

export function formatTokens(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

export function formatUsd(n: number): string {
  return `$${n.toFixed(2)}`;
}
