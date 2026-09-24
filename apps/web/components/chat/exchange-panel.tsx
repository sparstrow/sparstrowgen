"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";
import type {
  Exchange,
  ExchangeLine,
  ExchangeReport,
  ExchangeSent,
  ProviderId,
} from "@/lib/chat-types";
import { useExchange } from "@/lib/queries";
import {
  commandLine,
  formatBytes,
  formatOffset,
  formatted,
  groupLines,
  utf8Bytes,
  type LineGroup,
} from "@/lib/exchange";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Status } from "@/components/ui/status";
import { ChoiceToggle } from "./choice-toggle";

/* One turn's exchange, opened: what was sent to the CLI, what it said it
   loaded, and every line it printed (docs/specs/2026-09-23-raw-exchange.md,
   design in docs/design/prototypes/Chat/raw-exchange.handoff.md).

   This is a check, not a reading view. Nothing is summarised away: runs of
   streaming fragments are collapsed into one row, and every line in them is one
   click from being shown exactly as printed. */

/** How much of a line a row shows. A tool result can be a whole file on one
 *  line, and laying out 300 KB to show its first 80 characters is waste. */
const PREVIEW = 400;

export function ExchangePanel({
  entryId,
  provider,
  running,
}: {
  entryId: string;
  provider: ProviderId;
  running: boolean;
}) {
  const q = useExchange(entryId, true);

  if (q.isPending) {
    return (
      <Trail>
        <Section title={`Sent to ${provider}`}>
          <Skeleton className="mb-1.5 h-3 w-11/12" />
          <Skeleton className="mb-1.5 h-3 w-2/5" />
          <Skeleton className="h-16 w-full rounded-lg" />
        </Section>
        <Section title="Received">
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="grid grid-cols-[52px_140px_1fr] gap-2.5 py-1">
              <Skeleton className="h-3" />
              <Skeleton className="h-3" />
              <Skeleton className="h-3" />
            </div>
          ))}
        </Section>
      </Trail>
    );
  }

  if (q.isError) {
    return (
      <Trail>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 font-sans text-[13px] leading-5">
          <Status tone="danger">Could not load this exchange.</Status>
          <span className="text-muted-foreground">
            {(q.error as Error).message} The saved answer below is unaffected.
          </span>
          <Button variant="outline" size="xs" onClick={() => void q.refetch()}>
            Try again
          </Button>
        </div>
      </Trail>
    );
  }

  const ex = q.data;
  if (!ex.recorded || !ex.sent) {
    return (
      <Trail>
        <p className="font-sans text-xs leading-[18px] text-muted-foreground">
          {running
            ? `Nothing recorded yet. ${provider} has not been handed anything so far.`
            : "Not recorded. Only the saved answer below exists for this turn."}
        </p>
      </Trail>
    );
  }

  return (
    <Trail>
      <Sent provider={provider} sent={ex.sent} />
      <Loaded provider={provider} sent={ex.sent} report={ex.report} running={running} />
      <Received provider={provider} exchange={ex} running={running} />
    </Trail>
  );
}

function Trail({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-3 ml-px flex flex-col gap-5 border-l border-border pl-4">{children}</div>
  );
}

function Section({
  title,
  meta,
  action,
  children,
}: {
  title: React.ReactNode;
  meta?: React.ReactNode;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section>
      <h4 className="mb-2 flex items-baseline gap-2 font-sans text-[13px] leading-5 font-medium tracking-tight">
        {title}
        {meta && <span className="font-mono text-xs font-normal tracking-normal text-muted-foreground">{meta}</span>}
        {action && <span className="ml-auto">{action}</span>}
      </h4>
      {children}
    </section>
  );
}

function Facts({ children }: { children: React.ReactNode }) {
  return (
    <dl className="grid grid-cols-[88px_minmax(0,1fr)] gap-x-3 gap-y-1 text-xs leading-[18px]">
      {children}
    </dl>
  );
}

function Fact({ name, children }: { name: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{name}</dt>
      <dd className="m-0 [overflow-wrap:anywhere]">{children}</dd>
    </>
  );
}

// ---------------------------------------------------------------------------
// sent
// ---------------------------------------------------------------------------

function Sent({ provider, sent }: { provider: ProviderId; sent: ExchangeSent }) {
  const [view, setView] = useState<"prompt" | "exact">("prompt");
  const shown = view === "exact" && sent.launched ? sent.stdin : sent.prompt;
  return (
    <Section
      title={`Sent to ${provider}`}
      meta={sent.launched ? undefined : "· never started"}
    >
      <Facts>
        <Fact name="command">
          {sent.launched ? commandLine(sent) : <span className="text-muted-foreground">not run</span>}
        </Fact>
        <Fact name="folder">{sent.cwd || <span className="text-muted-foreground">not given</span>}</Fact>
        <Fact name="session">
          {sent.resumeSessionId ? (
            <>continued {sent.resumeSessionId}</>
          ) : (
            <span className="text-muted-foreground">new, nothing resumed</span>
          )}
        </Fact>
      </Facts>
      <div className="mt-2 overflow-hidden rounded-lg border border-border bg-card">
        <div className="flex items-center gap-2 border-b border-border py-1 pr-1.5 pl-2.5 text-xs text-muted-foreground">
          <span className="min-w-0 truncate">
            {sent.launched
              ? `stdin · ${formatBytes(utf8Bytes(sent.stdin))} · ${sent.prompt.length.toLocaleString()} characters of prompt`
              : `what would have been sent · ${sent.prompt.length.toLocaleString()} characters`}
          </span>
          {sent.launched && (
            <ChoiceToggle
              className="ml-auto"
              label="How to show what was sent"
              options={[
                { id: "prompt", label: "As the agent reads it" },
                { id: "exact", label: "Exact bytes" },
              ]}
              value={view}
              onChange={setView}
            />
          )}
        </div>
        <pre className="m-0 max-h-[360px] overflow-auto px-2.5 py-2 text-[12.5px] leading-[19px] whitespace-pre-wrap [overflow-wrap:anywhere]">
          {shown}
        </pre>
      </div>
    </Section>
  );
}

// ---------------------------------------------------------------------------
// what the CLI said it loaded
// ---------------------------------------------------------------------------

/** What a report can say about the CLI itself, in the order it is shown. The
 *  ones a CLI left out are named, so "codex does not say" is never read as
 *  "codex has none". */
const SELF: { key: keyof ExchangeReport; name: string }[] = [
  { key: "cliVersion", name: "version" },
  { key: "model", name: "model" },
  { key: "tools", name: "tools" },
  { key: "skills", name: "skills" },
];

function Loaded({
  provider,
  sent,
  report,
  running,
}: {
  provider: ProviderId;
  sent: ExchangeSent;
  report?: ExchangeReport;
  running: boolean;
}) {
  const r = report ?? {};
  const title = `What ${provider} reported loading`;
  if (!sent.launched) {
    return (
      <Section title={title}>
        <p className="font-sans text-xs leading-[18px] text-muted-foreground">
          {provider} never started, so it reported nothing.
        </p>
      </Section>
    );
  }
  const said = Object.values(r).some((v) => v !== undefined && v !== "");
  if (!said) {
    return (
      <Section title={title}>
        <p className="font-sans text-xs leading-[18px] text-muted-foreground">
          {running
            ? `Nothing yet. ${provider} says what it loaded, if anything, as it starts.`
            : `${provider} reported nothing about what it loaded.`}
        </p>
      </Section>
    );
  }
  const missing = SELF.filter((f) => !r[f.key] || (Array.isArray(r[f.key]) && (r[f.key] as string[]).length === 0)).map((f) => f.name);
  return (
    <Section title={title}>
      <Facts>
        {r.cliVersion && <Fact name="version">{provider} {r.cliVersion}</Fact>}
        {r.model && <Fact name="model">{r.model}</Fact>}
        {r.permissionMode && (
          <Fact name="permissions">
            {r.permissionMode} <span className="text-muted-foreground">mode</span>
          </Fact>
        )}
        {r.cwd && <Fact name="folder">{r.cwd}</Fact>}
        {r.sessionId && <Fact name="session">{r.sessionId}</Fact>}
        {r.tools && r.tools.length > 0 && <Fact name="tools">{tags(r.tools)}</Fact>}
        {r.skills && r.skills.length > 0 && <Fact name="skills">{tags(r.skills)}</Fact>}
        {r.agents && r.agents.length > 0 && <Fact name="agents">{tags(r.agents)}</Fact>}
        {r.cliVersion !== undefined && (
          // Only claude lists MCP servers, and it lists them even when there
          // are none, so "none" here is its answer rather than an absence.
          <Fact name="MCP servers">
            {r.mcpServers && r.mcpServers.length > 0 ? tags(r.mcpServers) : <span className="text-muted-foreground">none</span>}
          </Fact>
        )}
      </Facts>
      {missing.length > 0 && !running && (
        <p className="mt-2 font-sans text-xs leading-[18px] text-muted-foreground">
          {provider} did not report its {listWords(missing)}.
        </p>
      )}
      {r.usage ? (
        <Usage provider={provider} sent={sent} usage={r.usage} />
      ) : (
        !running && (
          <p className="mt-2 font-sans text-xs leading-[18px] text-muted-foreground">
            No token usage was reported for this turn.
          </p>
        )
      )}
    </Section>
  );
}

function Usage({
  provider,
  sent,
  usage,
}: {
  provider: ProviderId;
  sent: ExchangeSent;
  usage: NonNullable<ExchangeReport["usage"]>;
}) {
  const n = (x: number) => x.toLocaleString();
  const row = (name: string, value: number, part = false) => (
    <>
      <span className={part ? "pl-3 text-muted-foreground" : ""}>{name}</span>
      <span className={`text-right ${part ? "text-muted-foreground" : ""}`}>{n(value)}</span>
    </>
  );
  return (
    <>
      <div className="mt-3 grid w-max grid-cols-[max-content_max-content] gap-x-4 gap-y-0.5 text-xs leading-[18px] tabular-nums">
        {row("read", usage.input)}
        {usage.fromCache !== undefined && row("from cache", usage.fromCache, true)}
        {usage.toCache !== undefined && row("written to cache", usage.toCache, true)}
        {row("wrote", usage.output)}
        {usage.reasoning !== undefined && row("reasoning", usage.reasoning, true)}
        <span className="border-t border-border pt-0.5">total tokens</span>
        <span className="border-t border-border pt-0.5 text-right">{n(usage.total)}</span>
      </div>
      <p className="mt-2 max-w-xl font-sans text-xs leading-[18px] text-muted-foreground">
        You sent {n(sent.prompt.length)} characters. {provider} read {n(usage.input)} tokens,
        because every call to the model carries its own instructions and tool list, and
        everything said so far in the turn, not only your message.
      </p>
    </>
  );
}

function tags(xs: string[]) {
  return (
    <span className="flex flex-wrap gap-1">
      {xs.map((x) => (
        <Badge key={x} variant="outline" className="h-auto px-1.5 py-0 font-mono text-[11.5px] font-normal text-muted-foreground">
          {x}
        </Badge>
      ))}
    </span>
  );
}

function listWords(words: string[]): string {
  if (words.length < 2) return words.join("");
  return `${words.slice(0, -1).join(", ")} or ${words[words.length - 1]}`;
}

// ---------------------------------------------------------------------------
// received
// ---------------------------------------------------------------------------

function Received({
  provider,
  exchange,
  running,
}: {
  provider: ProviderId;
  exchange: Exchange;
  running: boolean;
}) {
  const { lines, sent, dropped } = exchange;
  const groups = useMemo(() => groupLines(lines), [lines]);
  const bytes = useMemo(() => lines.reduce((n, l) => n + utf8Bytes(l.text), 0), [lines]);

  async function copy() {
    try {
      // Exactly as received, one line each: pasted into a file it is the JSONL
      // the CLI printed, stdout and stderr interleaved in arrival order.
      await navigator.clipboard.writeText(lines.map((l) => l.text).join("\n"));
      toast.success(`Copied ${lines.length.toLocaleString()} lines`);
    } catch (err) {
      toast.error("Could not copy the lines", { description: (err as Error).message });
    }
  }

  return (
    <Section
      title="Received"
      meta={`${lines.length.toLocaleString()} lines · ${formatBytes(bytes)}`}
      action={
        lines.length > 0 && (
          <Button variant="ghost" size="xs" className="font-sans text-muted-foreground" onClick={() => void copy()}>
            Copy lines
          </Button>
        )
      }
    >
      {lines.length === 0 ? (
        <p className="font-sans text-xs leading-[18px] text-muted-foreground">
          {running ? (
            <Status tone="progress" quiet>
              Waiting for {provider}’s first line.
            </Status>
          ) : sent && !sent.launched ? (
            `Nothing. ${provider} was never started.`
          ) : (
            `Nothing. ${provider} printed no lines.`
          )}
        </p>
      ) : (
        <div className="flex flex-col">
          {groups.map((g) => (
            <GroupRow key={g.first} group={g} lines={lines} />
          ))}
        </div>
      )}
      {running && lines.length > 0 && (
        <p className="mt-1.5 font-sans text-xs leading-[18px]">
          <Status tone="progress" quiet>
            {provider} is still working. New lines appear as they arrive.
          </Status>
        </p>
      )}
      {dropped && (dropped.lines > 0 || dropped.bytes > 0) && (
        <p className="mt-1.5 font-sans text-xs leading-[18px]">
          <Status tone="warning">
            {dropped.lines > 0 ? "Recording stopped here: this turn’s record is full." : "Some over-long lines were cut."}
          </Status>{" "}
          <span className="text-muted-foreground">
            {formatBytes(dropped.bytes)}
            {dropped.lines > 0 && `, in ${dropped.lines.toLocaleString()} more lines,`} reached
            sparstrowgen and was not kept. The turn itself was not affected.
          </span>
        </p>
      )}
    </Section>
  );
}

const rowClass =
  "grid w-full grid-cols-[52px_minmax(0,max-content)_minmax(0,1fr)] items-baseline gap-x-2.5 rounded-md -ml-1.5 px-1.5 py-px text-left text-xs leading-5 hover:bg-accent aria-expanded:bg-muted";

function GroupRow({ group, lines }: { group: LineGroup; lines: ExchangeLine[] }) {
  const [open, setOpen] = useState(false);
  const first = lines[group.first];
  const kind = group.stream === "stderr" ? "stderr" : (group.label ?? "text");
  const preview = group.text !== null ? `“${group.text}”` : first.text;
  return (
    <>
      <button type="button" className={rowClass} aria-expanded={open} onClick={() => setOpen(!open)}>
        <span className="text-right text-muted-foreground tabular-nums">{formatOffset(first.atMs)}</span>
        <span className={`max-w-[36ch] truncate ${group.stream === "stderr" ? "font-medium" : ""}`}>
          {kind}
          {group.count > 1 && <span className="text-muted-foreground"> ×{group.count}</span>}
        </span>
        <span className="min-w-0 truncate text-muted-foreground">{preview.slice(0, PREVIEW)}</span>
      </button>
      {open && (
        <div className="mt-0.5 mb-1.5 ml-[62px]">
          {group.count === 1 ? (
            <LineBox line={first} index={group.first} />
          ) : (
            <div className="flex flex-col border-l border-border pl-2">
              {lines.slice(group.first, group.first + group.count).map((l, k) => (
                <SubRow key={l.seq} line={l} index={group.first + k} />
              ))}
            </div>
          )}
        </div>
      )}
    </>
  );
}

function SubRow({ line, index }: { line: ExchangeLine; index: number }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className={rowClass} aria-expanded={open} onClick={() => setOpen(!open)}>
        <span className="text-right text-muted-foreground tabular-nums">{formatOffset(line.atMs)}</span>
        <span>line {index + 1}</span>
        <span className="min-w-0 truncate text-muted-foreground">{line.text.slice(0, PREVIEW)}</span>
      </button>
      {open && (
        <div className="mt-0.5 mb-1.5">
          <LineBox line={line} index={index} />
        </div>
      )}
    </>
  );
}

function LineBox({ line, index }: { line: ExchangeLine; index: number }) {
  const [view, setView] = useState<"formatted" | "exact">("formatted");
  const pretty = useMemo(() => formatted(line.text), [line.text]);
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border py-1 pr-1.5 pl-2.5 text-xs text-muted-foreground">
        <span>
          line {index + 1} · {line.stream} · {formatBytes(utf8Bytes(line.text))}
        </span>
        {pretty !== null && (
          <ChoiceToggle
            className="ml-auto"
            label="How to show this line"
            options={[
              { id: "formatted", label: "Formatted" },
              { id: "exact", label: "Exact" },
            ]}
            value={view}
            onChange={setView}
          />
        )}
      </div>
      <pre className="m-0 max-h-[360px] overflow-auto px-2.5 py-2 text-[12.5px] leading-[19px] whitespace-pre-wrap [overflow-wrap:anywhere]">
        {view === "formatted" && pretty !== null ? pretty : line.text}
      </pre>
    </div>
  );
}
