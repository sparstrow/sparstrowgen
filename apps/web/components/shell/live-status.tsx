"use client";

import { Status, type StatusTone } from "@/components/ui/status";
import { useDaemon, useMachines } from "@/lib/queries";

/* The one line in the header that answers "if I send something now, will it
   run?".

   It reads two separate facts and never conflates them (docs/Decisions.md
   D-042, and the open question the prototype raised about what "live" means):

     this browser's socket   — down means we are being told nothing at all, so
                               the honest word is "Reconnecting", not a stale
                               "online" we have no way to disprove
     the computer            — only the server knows whether the daemon is
                               dialled in, and it says so over that socket

   The word is always shown beside the icon (D-040). The long form names the
   computer; below 768px only the word fits, so that is all it says. */

type Live = { tone: StatusTone; full: string; short: string };

function describe(
  serverConnected: boolean,
  online: boolean,
  tooOld: boolean,
  names: string[],
  known: boolean,
): Live {
  if (!serverConnected)
    return { tone: "progress", full: "Reconnecting…", short: "Reconnecting…" };
  if (known && names.length === 0)
    return { tone: "neutral", full: "No computer connected", short: "None" };

  // One computer is named; several are counted, because the server reports one
  // "is a computer reachable" fact for the account rather than one per machine,
  // so naming the first would be a guess about which. See the open question in
  // the shell handoff.
  const one = names.length === 1 ? names[0] : null;
  if (tooOld)
    return {
      tone: "warning",
      full: one ? `${one} needs an update` : "A computer needs an update",
      short: "Needs an update",
    };
  if (online)
    return {
      tone: "success",
      full: one ? `${one} online` : `${names.length} computers online`,
      short: "Online",
    };
  return {
    tone: "neutral",
    full: one ? `${one} offline` : "No computer online",
    short: "Offline",
  };
}

export function LiveStatus({ className }: { className?: string }) {
  const { online, tooOld, serverConnected } = useDaemon();
  const machines = useMachines();
  const names = (machines.data ?? []).map((m) => m.name);
  const live = describe(serverConnected, online, tooOld, names, machines.isSuccess);

  return (
    <Status tone={live.tone} size="sm" role="status" className={className}>
      <span className="hidden whitespace-nowrap md:inline">{live.full}</span>
      <span className="whitespace-nowrap md:hidden">{live.short}</span>
    </Status>
  );
}
