"use client";

import { useEffect, useState } from "react";
import { cn } from "cn";
import type { Model, ProviderId } from "@/lib/chat-types";
import { providerStyle } from "./provider-meta";
import { ProviderIcon } from "./provider-icon";

/* A turn in flight: the provider's header, a 3 by 3 pixel grid whose cells
   light in a travelling wavefront, a shimmering "working" and a live timer.
   Built from the design system's WorkingIndicator (the artifact linked from
   CLAUDE.md), which the owner asked for by name.

   It exists because codex sends no incremental text at all: on the one
   provider where a turn takes longest, nothing else on screen moves until the
   whole answer lands. So it animates only while work is happening, and is
   unmounted — not paused — when the turn ends, so it never pulses for nothing. */

export type PixelVariant = "drive" | "dots" | "orbit";

type Grid = { delays: (number | null)[]; dur: number; round: boolean };

/** Per-cell delays in ms. The cycle is shorter than the sweep, so two fronts
 *  are always in flight. null is a cell that stays dim — the orbit's centre. */
const GRIDS: Record<PixelVariant, Grid> = (() => {
  const chevron: number[] = [];
  const orbit: (number | null)[] = [];
  const perimeter = [0, 1, 2, 5, 8, 7, 6, 3];
  for (let i = 0; i < 9; i++) {
    const row = Math.floor(i / 3);
    const col = i % 3;
    chevron.push((col + Math.abs(row - 1)) * 90);
    const k = perimeter.indexOf(i);
    orbit.push(k === -1 ? null : k * 110);
  }
  return {
    drive: { delays: chevron, dur: 650, round: false },
    dots: { delays: chevron, dur: 650, round: true },
    orbit: { delays: orbit, dur: 950, round: false },
  };
})();

/** Which grid each agent gets, as the owner saw them side by side in the
 *  design system: square cells for claude, round for codex, a comet lapping
 *  the edge for agy. A provider this build does not know gets the default
 *  rather than nothing (AGENTS.md §3). */
const VARIANT: Partial<Record<ProviderId, PixelVariant>> = {
  claude: "drive",
  codex: "dots",
  agy: "orbit",
};

/** The bare grid, for a spot that has no provider to name. */
export function PixelLoader({
  variant = "drive",
  className,
}: {
  variant?: PixelVariant;
  className?: string;
}) {
  const grid = GRIDS[variant] ?? GRIDS.drive;
  return (
    <span className={cn("pixel-grid", className)} aria-hidden>
      {grid.delays.map((delay, i) => (
        <span
          key={i}
          className={cn("pixel-cell", grid.round && "is-round")}
          style={
            delay === null
              ? { opacity: 0.07, animation: "none" }
              : { animation: `pixel-on ${grid.dur}ms ease-in-out ${delay}ms infinite` }
          }
        />
      ))}
    </span>
  );
}

/** 12.0s, 1m 53.5s. Tenths are shown so the reading visibly moves. */
function formatElapsed(total: number): string {
  if (total < 60) return `${total.toFixed(1)}s`;
  return `${Math.floor(total / 60)}m ${(total % 60).toFixed(1)}s`;
}

export function WorkingIndicator({
  provider,
  model,
  startedAt,
  streams,
}: {
  provider: ProviderId;
  model: Model;
  /** When the turn was sent, in epoch ms. Timed from this rather than from
   *  mount, so opening another conversation and coming back does not start the
   *  count again at zero. */
  startedAt: number;
  streams: boolean;
}) {
  // The clock lives here, not in the chat surface: it ticks ten times a second
  // for the tenths, and re-rendering a whole transcript of markdown that often
  // to move one number would be the most expensive thing on the page.
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 100);
    return () => clearInterval(t);
  }, []);
  const seconds = Math.max(0, (now - startedAt) / 1000);
  const c = providerStyle(provider);

  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2">
        <ProviderIcon provider={provider} className={`size-4 ${c.text}`} />
        <span className={`text-sm font-medium ${c.text}`}>{provider}</span>
        <span className="text-xs text-muted-foreground">{model.label}</span>
      </div>
      <div className="flex items-center gap-2.5">
        <PixelLoader variant={VARIANT[provider]} />
        {/* Only the word is announced. The timer is hidden from assistive
            technology, or it would be read out ten times a second. */}
        <span className="working-shimmer text-[13px] leading-5 font-medium" role="status" aria-live="polite">
          working
        </span>
        <span className="font-mono text-xs text-muted-foreground tabular-nums" aria-hidden>
          {formatElapsed(seconds)}
        </span>
        {!streams && (
          // Says why nothing is appearing, on the provider where nothing will
          // until the end.
          <span className="text-xs text-muted-foreground/70">
            <span className="mr-2.5" aria-hidden>
              ·
            </span>
            {provider} sends its reply in one piece
          </span>
        )}
      </div>
    </div>
  );
}
