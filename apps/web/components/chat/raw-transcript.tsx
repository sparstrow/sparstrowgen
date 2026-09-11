"use client";

import type { Entry } from "@/lib/chat-types";
import { providerStyle, formatTokens, formatUsd } from "./provider-meta";

/* The transcript with nothing done to it.
 *
 * This is a check on the renderer, not a second way to read a conversation. A
 * markdown renderer decides what to show, and anything it does not understand
 * it drops silently — you cannot tell a reply that had no table from one whose
 * table went missing. Here the stored string is printed verbatim, so the two
 * views can be compared and the difference is entirely the renderer's doing.
 *
 * What this shows is exactly what markdown.tsx is given: the text as it is held
 * in our database. It is NOT the CLI's event stream — the reasoning and
 * tool-call events a provider emits alongside its answer are not stored at all,
 * so nothing here can reveal them (docs/Later.md L-10).
 *
 * Monospace because the content is literal: fence markers and indentation only
 * mean anything if they land where the agent put them. */

function Line({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-1 flex flex-wrap items-baseline gap-x-2 text-xs text-muted-foreground">
      {children}
    </div>
  );
}

function Body({ text }: { text: string }) {
  return (
    <pre className="whitespace-pre-wrap break-words text-[13px] leading-relaxed text-foreground">
      {text}
    </pre>
  );
}

export function RawTranscript({ entries }: { entries: Entry[] }) {
  return (
    <div className="space-y-6 font-mono">
      {entries.map((e) => {
        if (e.role === "user") {
          return (
            <div key={e.id}>
              <Line>
                <span className="text-foreground">you</span>
                <span>{e.at}</span>
              </Line>
              <Body text={e.text} />
            </div>
          );
        }

        if (e.role === "replay") {
          return (
            <Line key={e.id}>
              <span>switched to</span>
              <span className={providerStyle(e.to).text}>{e.to}</span>
              <span>{e.toModel.label}</span>
              <span>· replayed {e.messagesReplayed} messages</span>
              <span>· {formatTokens(e.tokens)} tokens</span>
            </Line>
          );
        }

        return (
          <div key={e.id}>
            <Line>
              <span className={providerStyle(e.provider).text}>{e.provider}</span>
              <span>{e.model.label}</span>
              <span>{e.at}</span>
              {e.usage && (
                <span>
                  · {formatTokens(e.usage.tokens)} tokens
                  {e.usage.usd !== undefined && <> · {formatUsd(e.usage.usd)}</>}
                </span>
              )}
            </Line>
            {e.text && <Body text={e.text} />}
            {e.stopped && (
              <p className="mt-1 text-[13px] text-muted-foreground">
                stopped{e.text ? "" : " before anything arrived"}
              </p>
            )}
            {e.failure && !e.stopped && (
              <p className="mt-1 text-[13px] text-destructive">
                turn did not finish: {e.failure}
              </p>
            )}
          </div>
        );
      })}
    </div>
  );
}
