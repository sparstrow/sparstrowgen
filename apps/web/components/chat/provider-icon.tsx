import type { ProviderId } from "@/lib/chat-types";

/* Provider marks, drawn rather than imported.
 *
 * Two constraints decided this. They must tint from `currentColor` so a single
 * component works in the provider colour, muted, and destructive contexts
 * without a variant per case — which rules out the vendors' own multi-colour
 * SVGs. And DESIGN.md forbids a hardcoded colour anywhere, which rules out
 * agy's rainbow. So all three are monochrome and inherit.
 *
 * These are recognisable approximations of each vendor's mark, not the official
 * assets. Consistent weight across the three matters more here than fidelity to
 * any one of them; drop in official SVGs later if it is worth it, but do it for
 * all three at once or the odd one out will look broken.
 */

type Props = {
  provider: ProviderId;
  className?: string;
};

/** Claude's starburst — ten tapered petals with rounded tips. */
function ClaudeMark({ className }: { className?: string }) {
  const petals = Array.from({ length: 10 }, (_, i) => i * 36);
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden>
      {petals.map((deg) => (
        <path
          key={deg}
          transform={`rotate(${deg} 12 12)`}
          d="M12 12 L10.85 5.2 A1.15 1.15 0 0 1 13.15 5.2 Z"
        />
      ))}
    </svg>
  );
}

/** The OpenAI knot, read as a six-lobed rosette — three loops at 60°. */
function CodexMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      className={className}
      aria-hidden
    >
      {[0, 60, 120].map((deg) => (
        <ellipse
          key={deg}
          cx="12"
          cy="12"
          rx="4"
          ry="8.6"
          transform={`rotate(${deg} 12 12)`}
        />
      ))}
    </svg>
  );
}

/** Antigravity's arch, monochrome — concentric bands over a flat base. */
function AgyMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      className={className}
      aria-hidden
    >
      <path d="M3.5 19.5 A8.5 8.5 0 0 1 20.5 19.5" />
      <path d="M8 19.5 A4 4 0 0 1 16 19.5" />
    </svg>
  );
}

const marks: Record<ProviderId, (p: { className?: string }) => React.ReactNode> =
  {
    claude: ClaudeMark,
    codex: CodexMark,
    agy: AgyMark,
  };

export function ProviderIcon({ provider, className = "size-4" }: Props) {
  const Mark = marks[provider];
  return <Mark className={className} />;
}
