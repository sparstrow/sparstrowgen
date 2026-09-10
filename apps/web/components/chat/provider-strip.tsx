"use client";

import { useEffect, useState } from "react";
import { Clock } from "lucide-react";
import type { Headroom, Provider } from "@/lib/chat-types";
import { providerClasses } from "./provider-meta";
import { ProviderIcon } from "./provider-icon";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

/* The honesty surface.
   Only claude reports a usage window at all, and what it reports is when the
   window RESETS — not how much is left. There is no percentage in the payload.
   An earlier version showed one behind a capacity bar; it was invented, and a
   bar here would be read as capacity we cannot measure. See D-018. */

/** "4h 12m", "38m", "now". Recomputed on a timer so it stays true without the
 *  server pushing a new value every minute. */
function untilReset(resetsAt: number, now: number): string {
  const secs = resetsAt - Math.floor(now / 1000);
  if (secs <= 0) return "now";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m`;
  return "under a minute";
}

/** "five_hour" → "5-hour". The provider's own word for the window, made
 *  readable without pretending to know more than it said. */
function windowLabel(window: string): string {
  const named: Record<string, string> = {
    five_hour: "5-hour",
    seven_day: "7-day",
    opus_seven_day: "7-day (Opus)",
  };
  return named[window] ?? window.replace(/_/g, " ");
}

function HeadroomReadout({ head, now }: { head: Headroom; now: number }) {
  const blocked = head.status !== "allowed";
  return (
    <span className="flex items-center gap-1.5">
      <Clock
        className={`size-3 ${blocked ? "text-capacity-out" : "text-muted-foreground"}`}
        aria-hidden
      />
      <span className="text-xs text-muted-foreground">
        {blocked && <span className="text-capacity-out">{head.status} · </span>}
        {windowLabel(head.window)} resets{" "}
        <span className="tabular-nums text-foreground">
          {untilReset(head.resetsAt, now)}
        </span>
      </span>
    </span>
  );
}

function ProviderChip({ provider, now }: { provider: Provider; now: number }) {
  const c = providerClasses[provider.id];
  const blocked = provider.availability === "blocked";
  const waiting = provider.availability === "waitable";
  const head = provider.headroom;

  const chip = (
    <div
      className={`flex min-w-0 items-center gap-2.5 rounded-md px-3 py-1.5 ${
        blocked || waiting ? "opacity-55" : ""
      }`}
    >
      <ProviderIcon
        provider={provider.id}
        className={`size-4 shrink-0 ${c.text}`}
      />
      <span className="text-sm font-medium">{provider.label}</span>

      {blocked || waiting ? (
        <span className="truncate text-xs text-muted-foreground">
          {provider.unavailableReason}
        </span>
      ) : head ? (
        <HeadroomReadout head={head} now={now} />
      ) : (
        <span className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            {provider.reportsLimits ? "limit unknown yet" : "no limit data"}
          </span>
          <span
            className="h-1 w-10 rounded-full border-t border-dashed border-capacity-unknown"
            aria-hidden
          />
        </span>
      )}
    </div>
  );

  const explanation = blocked
    ? `${provider.label} is installed but cannot run: ${provider.unavailableReason?.toLowerCase()}.`
    : waiting
      ? `${provider.label} can't be reached right now: ${provider.unavailableReason?.toLowerCase()}. Nothing is broken — it should come back on its own.`
      : head
        ? `${provider.label} reports a ${windowLabel(head.window)} usage window, currently ${head.status}, resetting in ${untilReset(head.resetsAt, now)}. It does not say how much of the window is left — only when it starts again.`
        : provider.reportsLimits
          ? `${provider.label} reports its usage window, but only while answering. Send one message and the reset time will appear here.`
          : `${provider.label} does not report usage limits at all. This is not the same as having plenty left — we genuinely cannot tell.`;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div
            tabIndex={0}
            className="cursor-default rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        }
      >
        {chip}
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">{explanation}</TooltipContent>
    </Tooltip>
  );
}

export function ProviderStrip({ providers }: { providers: Provider[] }) {
  // One clock for the whole strip. A minute is the right granularity for a
  // multi-hour window, and it keeps this from re-rendering every second.
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 30_000);
    return () => clearInterval(t);
  }, []);

  return (
    <div className="flex shrink-0 flex-wrap items-center gap-1 border-b px-3 py-2">
      {providers.map((p) => (
        <ProviderChip key={p.id} provider={p} now={now} />
      ))}
    </div>
  );
}
