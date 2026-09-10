"use client";

import type { Provider } from "@/lib/chat-types";
import { providerClasses, capacityClass } from "./provider-meta";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

/* The honesty surface. Three of four providers report no limit signal at all,
   so this must distinguish "plenty left" from "we cannot know" — showing an
   empty or full bar for an unknown would be a lie the backend cannot back up.
   See docs/Capabilities.md. */

function ProviderChip({ provider }: { provider: Provider }) {
  const c = providerClasses[provider.id];
  const blocked = provider.availability === "blocked";
  const head = provider.headroom;

  const chip = (
    <div
      className={`flex min-w-0 items-center gap-2.5 rounded-md px-3 py-1.5 ${
        blocked ? "opacity-55" : ""
      }`}
    >
      <span className={`size-2 shrink-0 rounded-full ${c.dot}`} aria-hidden />
      <span className="text-sm font-medium">{provider.label}</span>

      {blocked ? (
        <span className="truncate text-xs text-muted-foreground">
          {provider.unavailableReason}
        </span>
      ) : head ? (
        <span className="flex items-center gap-2">
          <span className="text-xs tabular-nums text-muted-foreground">
            {head.pct}%
          </span>
          <span className="h-1 w-16 overflow-hidden rounded-full bg-muted">
            <span
              className={`block h-full rounded-full ${capacityClass(head.pct)}`}
              style={{ width: `${head.pct}%` }}
            />
          </span>
          <span className="text-xs text-muted-foreground">
            resets {head.resetsIn}
          </span>
        </span>
      ) : (
        <span className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">no limit data</span>
          <span
            className="h-1 w-16 rounded-full border-t border-dashed border-capacity-unknown"
            aria-hidden
          />
        </span>
      )}
    </div>
  );

  const explanation = blocked
    ? `${provider.label} is installed but cannot run: ${provider.unavailableReason?.toLowerCase()}. Sign in to enable it.`
    : head
      ? `${provider.label} has ${head.pct}% of its ${head.window.replace("_", "-")} window left. Resets in ${head.resetsIn}.`
      : `${provider.label} does not report usage limits. This is not the same as having plenty left — we genuinely cannot tell.`;

  return (
    <Tooltip>
      <TooltipTrigger
        render={<div tabIndex={0} className="cursor-default outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md" />}
      >
        {chip}
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">{explanation}</TooltipContent>
    </Tooltip>
  );
}

export function ProviderStrip({ providers }: { providers: Provider[] }) {
  return (
    <div className="flex shrink-0 flex-wrap items-center gap-1 border-b px-3 py-2">
      {providers.map((p) => (
        <ProviderChip key={p.id} provider={p} />
      ))}
    </div>
  );
}
