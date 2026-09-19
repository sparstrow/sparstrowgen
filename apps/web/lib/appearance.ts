import type { Appearance } from "./api";

/* The appearance a person chose, as the page needs it: the named choices, and
   the two jobs that must work before React runs — reading the last known
   appearance and putting it on the document.

   The account is the source of truth (spec 2026-09-12-appearance-preferences).
   The copy in this browser exists only so the first paint is not the wrong
   theme; it is overwritten by the account's answer as soon as that arrives. */

export const DEFAULT_APPEARANCE: Appearance = { mode: "system", surface: "mono", accent: "neutral" };

export const MODES: { id: Appearance["mode"]; label: string; description: string }[] = [
  { id: "light", label: "Light", description: "Always the light expression." },
  { id: "dark", label: "Dark", description: "Always the dark expression." },
  { id: "system", label: "System", description: "Follow this computer's setting." },
];

export const SURFACES: { id: Appearance["surface"]; label: string; description: string }[] = [
  { id: "paper", label: "Paper", description: "Warm, with a linen cast." },
  { id: "slate", label: "Slate", description: "Cool blue-grey." },
  { id: "soft", label: "Soft", description: "Lifted and gentle, with less contrast." },
  { id: "mono", label: "Mono", description: "Pure neutral, no colour cast." },
];

/** The swatch colours are the accent's own dark-mode value, so the row of
 *  choices reads as the colours themselves rather than as five grey chips. */
export const ACCENTS: { id: Appearance["accent"]; label: string; swatch: string }[] = [
  { id: "neutral", label: "Neutral", swatch: "oklch(0.5 0 0)" },
  { id: "amber", label: "Amber", swatch: "oklch(0.78 0.15 70)" },
  { id: "violet", label: "Violet", swatch: "oklch(0.78 0.18 285)" },
  { id: "blue", label: "Blue", swatch: "oklch(0.78 0.16 250)" },
  { id: "teal", label: "Teal", swatch: "oklch(0.78 0.12 190)" },
  { id: "rose", label: "Rose", swatch: "oklch(0.78 0.16 15)" },
];

export const APPEARANCE_KEY = "sparstrowgen.appearance";

function isOneOf<T extends string>(value: unknown, allowed: readonly { id: T }[]): value is T {
  return typeof value === "string" && allowed.some((a) => a.id === value);
}

/** Replaces anything this version does not understand, part by part — the same
 *  rule the server applies when reading a saved choice. */
export function readable(value: unknown): Appearance {
  const raw = (value ?? {}) as Partial<Appearance>;
  return {
    mode: isOneOf(raw.mode, MODES) ? raw.mode : DEFAULT_APPEARANCE.mode,
    surface: isOneOf(raw.surface, SURFACES) ? raw.surface : DEFAULT_APPEARANCE.surface,
    accent: isOneOf(raw.accent, ACCENTS) ? raw.accent : DEFAULT_APPEARANCE.accent,
  };
}

export function sameAppearance(a: Appearance, b: Appearance) {
  return a.mode === b.mode && a.surface === b.surface && a.accent === b.accent;
}

/** Puts the surface and accent on the document. The mode is next-themes' job,
 *  because it owns the `dark` class and the system-preference listener. */
export function applyAppearance(appearance: Appearance) {
  if (typeof document === "undefined") return;
  const html = document.documentElement;
  html.dataset.surface = appearance.surface;
  html.dataset.accent = appearance.accent;
}

/** Remembers the appearance for the next first paint. Storage can throw — a
 *  private window, or site data blocked — and a theme is never worth an error. */
export function cacheAppearance(appearance: Appearance) {
  try {
    localStorage.setItem(APPEARANCE_KEY, JSON.stringify(appearance));
  } catch {
    // The account still has it; only this browser's head start is lost.
  }
}

export function cachedAppearance(): Appearance {
  try {
    return readable(JSON.parse(localStorage.getItem(APPEARANCE_KEY) ?? "null"));
  } catch {
    return DEFAULT_APPEARANCE;
  }
}

/** Runs before React, in a <script> in the document head, so the first paint is
 *  already the right theme. It is inlined as a string because it must not wait
 *  for any bundle. next-themes writes the `dark` class the same way.
 *
 *  Kept deliberately small and forgiving: any failure leaves the document's
 *  built-in defaults in place, which are readable. */
export const FIRST_PAINT_SCRIPT = `(function(){try{
var a=JSON.parse(localStorage.getItem(${JSON.stringify(APPEARANCE_KEY)})||"null")||{};
var s=["paper","slate","soft","mono"].indexOf(a.surface)>=0?a.surface:"mono";
var c=["neutral","amber","violet","blue","teal","rose"].indexOf(a.accent)>=0?a.accent:"neutral";
document.documentElement.dataset.surface=s;document.documentElement.dataset.accent=c;
}catch(e){document.documentElement.dataset.surface="mono";document.documentElement.dataset.accent="neutral";}})();`;
