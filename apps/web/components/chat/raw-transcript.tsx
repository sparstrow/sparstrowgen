"use client";

import { useState } from "react";
import { ChevronRight, CircleMinus } from "lucide-react";
import type { AgentMessage, Entry, ExchangeSummary } from "@/lib/chat-types";
import { useExchangeSummaries } from "@/lib/queries";
import { formatBytes, formatDuration } from "@/lib/exchange";
import { providerStyle, formatTokens, formatUsd, clockTime } from "./provider-meta";
import { Status } from "@/components/ui/status";
import { ExchangePanel } from "./exchange-panel";

/* The transcript with nothing done to it, and behind each agent turn the whole
 * exchange with its CLI.
 *
 * Two checks in one place. The saved text is printed verbatim, which is exactly
 * what markdown.tsx is given, so any difference from the Rendered view is the
 * renderer's doing. And each turn's exchange shows what was actually sent to the
 * agent, what it said it loaded, and every line it printed back — so a wrong
 * answer can be told apart from a wrong rendering of a right one
 * (docs/specs/2026-09-23-raw-exchange.md).
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

export function RawTranscript({
  conversationId,
  entries,
  runningId,
}: {
  conversationId: string;
  entries: Entry[];
  /** The agent entry a turn is streaming into right now, if any. */
  runningId: string | null;
}) {
  const summaries = useExchangeSummaries(conversationId, true);
  const byEntry = new Map((summaries.data ?? []).map((s) => [s.entryId, s]));
  // The latest turn opens by itself: "what just happened" is the usual question.
  const latest = [...entries].reverse().find((e) => e.role === "agent")?.id ?? null;

  return (
    <div className="space-y-6 font-mono">
      {entries.map((e) => {
        if (e.role === "user") {
          return (
            <div key={e.id}>
              <Line>
                <span className="text-foreground">you</span>
                <span>{clockTime(e.at)}</span>
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
          <AgentTurn
            key={e.id}
            entry={e}
            running={e.id === runningId}
            summary={byEntry.get(e.id)}
            // Until the summaries answer, a turn with no record cannot be told
            // from one whose record has not been asked about yet.
            known={summaries.isSuccess}
            startOpen={e.id === latest}
          />
        );
      })}
    </div>
  );
}

function AgentTurn({
  entry: e,
  running,
  summary,
  known,
  startOpen,
}: {
  entry: AgentMessage;
  running: boolean;
  summary?: ExchangeSummary;
  known: boolean;
  startOpen: boolean;
}) {
  const [open, setOpen] = useState(startOpen);
  const unrecorded = known && !summary && !running;

  return (
    <div>
      <Line>
        <span className={providerStyle(e.provider).text}>{e.provider}</span>
        <span>{e.model.label}</span>
        <span>{clockTime(e.at)}</span>
        {e.usage && (
          <span>
            · {formatTokens(e.usage.tokens)} tokens
            {e.usage.usd !== undefined && <> · {formatUsd(e.usage.usd)}</>}
          </span>
        )}
      </Line>

      {unrecorded ? (
        <p className="mt-0.5 mb-2 flex items-start gap-1.5 text-xs leading-4 text-muted-foreground">
          <CircleMinus className="mt-px size-3.5 shrink-0" aria-hidden />
          <span>
            Not recorded. This turn ran before your computer’s sparstrowgen kept a record of what
            it sent and received, or it never reached your computer, so only the saved answer
            below exists.
          </span>
        </p>
      ) : (
        <>
          <button
            type="button"
            onClick={() => setOpen(!open)}
            aria-expanded={open}
            className="-ml-1.5 mt-0.5 mb-2 inline-flex items-center gap-1.5 rounded-md px-1.5 py-0.5 text-xs leading-4 text-muted-foreground transition-colors duration-150 ease-out hover:bg-accent hover:text-foreground"
          >
            <ChevronRight
              className="size-3.5 shrink-0 transition-transform duration-180 ease-[cubic-bezier(0.25,1,0.5,1)] motion-reduce:transition-none [[aria-expanded=true]>&]:rotate-90"
              aria-hidden
            />
            <span>{toggleLabel(e, summary, running)}</span>
          </button>
          {open && <ExchangePanel entryId={e.id} provider={e.provider} running={running} />}
        </>
      )}

      {/* Only over text: a turn that ended with a failure and no answer has
          nothing saved to label, and the failure below speaks for itself. */}
      {open && !unrecorded && e.text && (
        <div className="mb-1 text-xs text-muted-foreground">saved answer · what Rendered shows</div>
      )}
      {e.text && <Body text={e.text} />}
      {e.stopped && (
        <p className="mt-1 text-[13px] text-muted-foreground">
          stopped{e.text ? "" : " before anything arrived"}
        </p>
      )}
      {e.failure && !e.stopped && (
        <p className="mt-1 text-[13px]">
          <Status tone="danger">turn did not finish: {e.failure}</Status>
        </p>
      )}
    </div>
  );
}

function toggleLabel(e: AgentMessage, s: ExchangeSummary | undefined, running: boolean): string {
  if (!s) return running ? "sent and received · starting" : "sent and received";
  if (!s.launched) return `sent · ${e.provider} never started`;
  const size = `${s.lines.toLocaleString()} lines · ${formatBytes(s.bytes)}`;
  return running ? `sent and received · ${size} so far` : `sent and received · ${size} · ${formatDuration(s.lastAtMs)}`;
}
