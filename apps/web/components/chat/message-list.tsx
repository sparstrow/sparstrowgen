"use client";

import { AlertTriangle, ArrowRightLeft, CircleSlash } from "lucide-react";
import type { Entry, Model, ProviderId } from "@/lib/chat-types";
import { providerClasses, formatTokens, formatUsd } from "./provider-meta";
import { ProviderIcon } from "./provider-icon";
import { Markdown } from "./markdown";
import { Skeleton } from "@/components/ui/skeleton";

/* The owner's messages sit right, as a bubble. Agent output runs down the
   centre as a reading column with no bubble at all — it is usually long, often
   contains code, and is the thing actually being read. Giving it a container
   would only narrow it. */

function UserBubble({ text, at }: { text: string; at: string }) {
  return (
    <div className="flex justify-end">
      <div className="max-w-[80%]">
        <div className="rounded-2xl rounded-br-md bg-mine px-4 py-2.5 text-[15px] leading-relaxed text-mine-foreground">
          {text}
        </div>
        <div className="mt-1 pr-1 text-right text-xs text-muted-foreground">
          {at}
        </div>
      </div>
    </div>
  );
}

function AgentTurn({
  provider,
  model,
  text,
  at,
  usage,
  failure,
  stopped,
  streaming,
}: {
  provider: ProviderId;
  model: Model;
  text: string;
  at: string;
  usage?: { tokens: number; usd?: number };
  failure?: string;
  stopped?: boolean;
  streaming?: boolean;
}) {
  const c = providerClasses[provider];
  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2">
        <ProviderIcon provider={provider} className={`size-4 ${c.text}`} />
        <span className={`text-sm font-medium ${c.text}`}>{provider}</span>
        <span className="text-xs text-muted-foreground">{model.label}</span>
        <span className="text-xs text-muted-foreground">· {at}</span>
      </div>

      {/* Rendered as markdown, because that is what every provider actually
          sends. A dedicated `code` field could only ever hold one block, and
          real answers interleave prose with several. */}
      <Markdown text={text} />
      {streaming && (
        <span className="-mt-1 ml-0.5 inline-block h-4 w-1.5 translate-y-0.5 rounded-xs bg-foreground/60" />
      )}

      {/* Deliberately not the destructive treatment below. Stopping a turn is
          something the owner chose to do, and putting a red alert box around a
          deliberate act reads as though something went wrong. */}
      {stopped && (
        <div className="mt-3 flex items-start gap-2.5 rounded-lg border border-dashed bg-muted/40 px-3.5 py-2.5">
          <CircleSlash
            className="mt-0.5 size-4 shrink-0 text-muted-foreground"
            aria-hidden
          />
          <div className="text-sm">
            <p className="font-medium">You stopped this</p>
            <p className="mt-0.5 text-muted-foreground">
              {text
                ? "What had arrived is kept above."
                : `Nothing had arrived yet — ${provider} sends its reply in one piece.`}
            </p>
          </div>
        </div>
      )}

      {/* A stopped turn's CLI often complains on its way out, and that
          complaint is a consequence of the stop rather than a reason worth
          reading. The stop is the truer account, so it is the one shown. */}
      {failure && !stopped && (
        <div className="mt-3 flex items-start gap-2.5 rounded-lg border border-destructive/40 bg-destructive/10 px-3.5 py-2.5">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-destructive" />
          <div className="text-sm">
            <p className="font-medium text-destructive">Turn did not finish</p>
            <p className="mt-0.5 text-muted-foreground">{failure}</p>
            <p className="mt-1 text-muted-foreground">
              What arrived before it stopped is kept above.
            </p>
          </div>
        </div>
      )}

      {usage && !streaming && (
        <div className="mt-2 font-mono text-xs text-muted-foreground">
          {formatTokens(usage.tokens)} tokens
          {usage.usd !== undefined && <> · {formatUsd(usage.usd)}</>}
        </div>
      )}
    </div>
  );
}

/* The switch marker. It is written into the transcript at the moment the replay
   was actually paid for — the first message sent after switching — never when
   the provider was selected. Choosing a provider costs nothing. */
function ReplayDivider({
  to,
  toModel,
  messagesReplayed,
  tokens,
  switched,
}: {
  to: ProviderId;
  toModel: Model;
  messagesReplayed: number;
  tokens: number;
  /** False when the same provider is being caught up rather than a new one
   *  taking over — which happens when a conversation moves to another folder
   *  and the provider sessions go with it. Saying "switched to claude" when
   *  nothing switched would be a small lie in the record. */
  switched: boolean;
}) {
  const c = providerClasses[to];
  return (
    <div className="flex items-center gap-3 py-1" role="separator">
      <span className="h-px flex-1 bg-border" />
      <span className="flex items-center gap-2 text-xs text-muted-foreground">
        <ArrowRightLeft className="size-3.5" aria-hidden />
        {switched ? "switched to" : "caught up"}
        <ProviderIcon provider={to} className={`size-3.5 ${c.text}`} />
        <span className={c.text}>{to}</span>
        <span className="text-muted-foreground/70">{toModel.label}</span>
        <span aria-hidden>·</span>
        replayed {messagesReplayed} messages
        <span aria-hidden>·</span>
        <span className="font-mono">{formatTokens(tokens)} tokens</span>
      </span>
      <span className="h-px flex-1 bg-border" />
    </div>
  );
}

/** Shown while a turn is in flight. Deliberately not tied to text arriving:
 *  codex emits no incremental output at all, so on that provider this is the
 *  only sign of life until the whole answer lands. */
export function WorkingIndicator({
  provider,
  model,
  elapsed,
  streams,
}: {
  provider: ProviderId;
  model: Model;
  elapsed: number;
  streams: boolean;
}) {
  const c = providerClasses[provider];
  return (
    <div>
      <div className="mb-1.5 flex items-center gap-2">
        <ProviderIcon provider={provider} className={`size-4 ${c.text}`} />
        <span className={`text-sm font-medium ${c.text}`}>{provider}</span>
        <span className="text-xs text-muted-foreground">{model.label}</span>
      </div>
      <div
        className="flex items-center gap-2.5 text-sm text-muted-foreground"
        role="status"
        aria-live="polite"
      >
        <span className="flex gap-1" aria-hidden>
          <span className="working-dot size-1.5 rounded-full bg-muted-foreground" />
          <span className="working-dot size-1.5 rounded-full bg-muted-foreground" />
          <span className="working-dot size-1.5 rounded-full bg-muted-foreground" />
        </span>
        <span className="tabular-nums">
          working · {elapsed}s
          {!streams && (
            <span className="text-muted-foreground/70">
              {" "}
              · {provider} sends its reply in one piece
            </span>
          )}
        </span>
      </div>
    </div>
  );
}

export function MessageSkeleton() {
  return (
    <div className="space-y-8" aria-busy="true" aria-label="Loading conversation">
      <div className="flex justify-end">
        <Skeleton className="h-11 w-2/5 rounded-2xl" />
      </div>
      <div className="space-y-2">
        <Skeleton className="h-3.5 w-32" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-11/12" />
        <Skeleton className="h-4 w-3/4" />
      </div>
      <div className="flex justify-end">
        <Skeleton className="h-11 w-1/3 rounded-2xl" />
      </div>
    </div>
  );
}

/** Which provider answered most recently before position `i`. Undefined at the
 *  start of a conversation, where a catch-up cannot be a switch. */
function lastAgentBefore(entries: Entry[], i: number): ProviderId | undefined {
  for (let j = i - 1; j >= 0; j--) {
    const e = entries[j];
    if (e.role === "agent") return e.provider;
  }
  return undefined;
}

export function MessageList({
  entries,
  streamingId,
}: {
  entries: Entry[];
  streamingId?: string | null;
}) {
  return (
    <div className="space-y-8">
      {entries.map((e, i) => {
        if (e.role === "user")
          return <UserBubble key={e.id} text={e.text} at={e.at} />;
        if (e.role === "replay")
          return (
            <ReplayDivider
              key={e.id}
              to={e.to}
              toModel={e.toModel}
              messagesReplayed={e.messagesReplayed}
              tokens={e.tokens}
              // Whether anything actually switched is not stored on the marker;
              // it is whether the agent that answered last was someone else.
              switched={lastAgentBefore(entries, i) !== e.to}
            />
          );
        return (
          <AgentTurn
            key={e.id}
            provider={e.provider}
            model={e.model}
            text={e.text}
            at={e.at}
            usage={e.usage}
            failure={e.failure}
            stopped={e.stopped}
            streaming={streamingId === e.id}
          />
        );
      })}
    </div>
  );
}
