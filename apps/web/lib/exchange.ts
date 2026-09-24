import type { ExchangeLine, ExchangeSent } from "./chat-types";

/* Reading a turn's record for display (docs/specs/2026-09-23-raw-exchange.md).

   Nothing here knows what a provider MEANS. A line is named by the keys it
   carries, so an event type no CLI had when this was written still shows up,
   named, rather than being filtered out. What a CLI said about itself is the
   server's reading of the lines (ExchangeReport); this file only arranges them. */

type Json = Record<string, unknown>;

type Reading = {
  /** "system · init", "content_block_delta · text_delta", or null for a line
   *  that is not a JSON object. */
  label: string | null;
  /** The text a streaming fragment carries, so a run of them reads as the
   *  sentence they spell. Null for anything that is not a fragment. */
  piece: string | null;
};

// Lines are appended, never replaced, so a line object read once is never read
// again: a turn that grows by a batch every quarter second re-reads only the
// batch, however long the record has become.
const readings = new WeakMap<ExchangeLine, Reading>();

function obj(v: unknown): Json | null {
  return v !== null && typeof v === "object" && !Array.isArray(v) ? (v as Json) : null;
}

function str(v: unknown): string | null {
  return typeof v === "string" ? v : null;
}

export function parseLine(text: string): Json | null {
  if (!text.startsWith("{")) return null;
  try {
    return obj(JSON.parse(text));
  } catch {
    return null;
  }
}

function read(line: ExchangeLine): Reading {
  const cached = readings.get(line);
  if (cached) return cached;
  const o = line.stream === "stdout" ? parseLine(line.text) : null;
  const reading: Reading = { label: o ? labelOf(o) : null, piece: o ? pieceOf(o) : null };
  readings.set(line, reading);
  return reading;
}

function labelOf(o: Json): string | null {
  // A wrapper that only says "an event follows" (claude's stream_event) is
  // named by the event inside it. The wrapper is still in the line.
  const ev = obj(o.event);
  if (ev) {
    const delta = obj(ev.delta);
    return [str(ev.type), delta && str(delta.type)].filter(Boolean).join(" · ") || null;
  }
  const kind = str(o.type) ?? str(o.event);
  const sub =
    str(o.subtype) ??
    str(obj(o.item)?.type) ??
    str(obj(o.step_update)?.state);
  return [kind, sub].filter(Boolean).join(" · ") || null;
}

function pieceOf(o: Json): string | null {
  const delta = obj(obj(o.event)?.delta);
  if (delta) return str(delta.text) ?? str(delta.thinking) ?? str(delta.partial_json);
  return str(obj(o.step_update)?.text_delta);
}

/** One row: a single line, or a run of streaming fragments of one kind. */
export type LineGroup = {
  /** Index into the record's lines of the first line in the group. */
  first: number;
  count: number;
  stream: ExchangeLine["stream"];
  label: string | null;
  /** The fragments joined, for a run; null for a single ordinary line. */
  text: string | null;
};

/** Consecutive fragments of one kind collapse into one row with a count;
 *  everything else is one row per line. Opening a group lists every line. */
export function groupLines(lines: ExchangeLine[]): LineGroup[] {
  const out: LineGroup[] = [];
  lines.forEach((line, i) => {
    const { label, piece } = read(line);
    const last = out[out.length - 1];
    if (piece !== null && last && last.text !== null && last.label === label && last.stream === line.stream) {
      last.count++;
      last.text += piece;
      return;
    }
    out.push({ first: i, count: 1, stream: line.stream, label, text: piece });
  });
  return out;
}

/** The command as it was run. An argument with a space or a quote in it is
 *  quoted, so where one argument ends is not left to guesswork. */
export function commandLine(sent: ExchangeSent): string {
  const quote = (a: string) => (a === "" || /[\s"]/.test(a) ? JSON.stringify(a) : a);
  return [sent.program, ...sent.args].map(quote).join(" ");
}

const encoder = new TextEncoder();

/** Bytes as the server counts them: UTF-8. */
export function utf8Bytes(text: string): number {
  return encoder.encode(text).length;
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatOffset(ms: number): string {
  return `${(ms / 1000).toFixed(2)}s`;
}

export function formatDuration(ms: number): string {
  const s = ms / 1000;
  return s < 60 ? `${s.toFixed(1)}s` : `${Math.floor(s / 60)}m ${Math.round(s % 60)}s`;
}

/** A line as JSON laid out to read, or null when it is not JSON. */
export function formatted(text: string): string | null {
  const o = parseLine(text);
  return o ? JSON.stringify(o, null, 2) : null;
}
