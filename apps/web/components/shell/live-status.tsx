"use client";

import { Status, type StatusTone } from "@/components/ui/status";
import { useDaemon, useMachines, useWorkspaceReach } from "@/lib/queries";

/* The one line in the header that answers "if I send something now, will it
   run?".

   It reads separate facts and never conflates them (docs/Decisions.md D-042,
   and the open question the prototype raised about what "live" means):

     this browser's socket   — down means we are being told nothing at all, so
                               the honest word is "Reconnecting", not a stale
                               "online" we have no way to disprove
     the computers           — only the server knows which are dialled in, and
                               it reports that per computer

   "Will it run" is asked of the workspace on screen, not the account: a
   workspace runs only on the computers it was given (00018), so the laptop
   being online says nothing about a workspace that has only the desktop.

   The word is always shown beside the icon (D-040). The long form names the
   computer, and only when the HEADER is wide enough for it — not the screen:
   with the rail and the conversation list open, a wide window still leaves a
   narrow header, and the name used to push the conversation title out of it
   (docs/Bugs.md B-51). Anywhere outside a header, only the word is shown. */

type Live = { tone: StatusTone; full: string; short: string };

function describe({
  serverConnected,
  tooOld,
  accountHasNone,
  workspaceHasNone,
  online,
  assigned,
  reachable,
}: {
  serverConnected: boolean;
  tooOld: boolean;
  accountHasNone: boolean;
  workspaceHasNone: boolean;
  online: string[];
  assigned: string[];
  reachable: boolean;
}): Live {
  if (!serverConnected)
    return { tone: "progress", full: "Reconnecting…", short: "Reconnecting…" };
  if (accountHasNone)
    return { tone: "neutral", full: "No computer connected", short: "No computer" };
  if (workspaceHasNone && !reachable)
    return { tone: "neutral", full: "No computer in this workspace", short: "No computer" };

  // Counted from the computers actually online. It used to count every paired
  // computer and call them all online (docs/Bugs.md B-49). One is named;
  // several are counted, since which of them takes the next turn is not yet
  // something the app shows (docs/KnownGaps.md G-44).
  if (tooOld)
    return {
      tone: "warning",
      full: online.length === 1 ? `${online[0]} needs an update` : "A computer needs an update",
      short: "Needs an update",
    };
  if (reachable)
    return {
      tone: "success",
      full:
        online.length === 1
          ? `${online[0]} online`
          : online.length > 1
            ? `${online.length} computers online`
            : "Online",
      short: "Online",
    };
  return {
    tone: "neutral",
    full: assigned.length === 1 ? `${assigned[0]} offline` : "No computer online",
    short: "Offline",
  };
}

export function LiveStatus({ className }: { className?: string }) {
  const { online: daemonOnline, tooOld, serverConnected } = useDaemon();
  const machines = useMachines();
  const reach = useWorkspaceReach();
  const live = describe({
    serverConnected,
    tooOld,
    accountHasNone: machines.isSuccess && machines.data.length === 0,
    workspaceHasNone: reach.known && reach.assigned.length === 0,
    online: reach.online.map((m) => m.name),
    assigned: reach.assigned.map((m) => m.name),
    // Until the workspace's computers have loaded, the account's answer is the
    // best one there is, and far better than a flash of "offline".
    reachable: reach.known ? reach.reachable : daemonOnline,
  });

  return (
    <Status tone={live.tone} size="sm" role="status" className={className}>
      <span className="hidden whitespace-nowrap @3xl/header:inline">{live.full}</span>
      <span className="whitespace-nowrap @3xl/header:hidden">{live.short}</span>
    </Status>
  );
}
