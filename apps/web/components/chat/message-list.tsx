"use client";

import { AlertTriangle, ArrowRightLeft } from "lucide-react";
import type { Entry, ProviderId } from "@/lib/chat-types";
import { providerClasses, formatTokens, formatUsd } from "./provider-meta";
import { Skeleton } from "@/components/ui/skeleton";

/* The owner's messages sit right, as a bubble. Agent output runs down the
   centre as a reading column with no bubble at all — it is usually long, often
   contains code, and is the thing actually being read. Giving it a container
   would only narrow it. */

function CodeBlock({ body }: { lang: string; body: string }) {
  return (
    <pre className="mt-3 overflow-x-auto rounded-lg border bg-background/60 p-3.5 text-[13px] leading-relaxed">
      <code className="font-mono">{body}</code>
    </pre>
  );
}

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
  code,
  failure,
  streaming,
}: {
  provider: ProviderId;
  model: string;
  text: string;
  at: string;
  usage?: { tokens: number; usd?: number };
  code?: { lang: string; body: string };
  failure?: string;
  streaming?: boolean;
}) {
  const c = providerClasses[provider];
  return (
    <div>
      <div className="mb-1.5 flex items-baseline gap-2">
        <span className={`size-2 rounded-full ${c.dot}`} aria-hidden />
        <span className={`text-sm font-medium ${c.text}`}>{provider}</span>
        <span className="text-xs text-muted-foreground">{model}</span>
        <span className="text-xs text-muted-foreground">· {at}</span>
      </div>

      <div className="text-[15px] leading-relaxed whitespace-pre-wrap">
        {text}
        {streaming && (
          <span className="ml-0.5 inline-block h-4 w-1.5 translate-y-0.5 rounded-xs bg-foreground/60" />
        )}
      </div>

      {code && <CodeBlock lang={code.lang} body={code.body} />}

      {failure && (
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
}: {
  to: ProviderId;
  toModel: string;
  messagesReplayed: number;
  tokens: number;
}) {
  const c = providerClasses[to];
  return (
    <div className="flex items-center gap-3 py-1" role="separator">
      <span className="h-px flex-1 bg-border" />
      <span className="flex items-center gap-2 text-xs text-muted-foreground">
        <ArrowRightLeft className="size-3.5" aria-hidden />
        switched to <span className={c.text}>{to}</span>
        <span className="text-muted-foreground/70">{toModel}</span>
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
  model: string;
  elapsed: number;
  streams: boolean;
}) {
  const c = providerClasses[provider];
  return (
    <div>
      <div className="mb-1.5 flex items-baseline gap-2">
        <span className={`size-2 rounded-full ${c.dot}`} aria-hidden />
        <span className={`text-sm font-medium ${c.text}`}>{provider}</span>
        <span className="text-xs text-muted-foreground">{model}</span>
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

export function MessageList({
  entries,
  streamingId,
}: {
  entries: Entry[];
  streamingId?: string | null;
}) {
  return (
    <div className="space-y-8">
      {entries.map((e) => {
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
            code={e.code}
            failure={e.failure}
            streaming={streamingId === e.id}
          />
        );
      })}
    </div>
  );
}
