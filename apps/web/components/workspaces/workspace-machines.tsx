"use client";

import Link from "next/link";
import { MonitorSmartphone } from "lucide-react";
import { cn } from "cn";
import type { AssignedMachine, AssignedWorkspace } from "@/lib/chat-types";
import {
  useMachineWorkspaces,
  useSetMachineWorkspace,
  useSetWorkspaceMachine,
  useWorkspaceMachines,
} from "@/lib/queries";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Status } from "@/components/ui/status";
import { Switch } from "@/components/ui/switch";

/* Which computers a workspace may use, and the same assignment from a
   computer's end (migration 00018). The owner asked for both directions: "I
   need to choose which machine needs added to that workspace or vice versa
   whick workspace needs to added to the machines."

   It is one row in one table, read from either side, so the two surfaces below
   cannot disagree — and both invalidate the other's query, so a change made in
   one is true in the other without a refresh.

   A switch rather than a checkbox: this turns something on and off rather than
   selecting it from a set, and the app already uses a switch for automatic
   updates, which is the same shape of choice. */

function Row({
  label,
  ariaLabel,
  hint,
  on,
  busy,
  onChange,
}: {
  label: React.ReactNode;
  ariaLabel: string;
  hint?: React.ReactNode;
  on: boolean;
  busy: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-4 py-3">
      <div className="flex min-w-0 items-center gap-3">
        <MonitorSmartphone className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{label}</div>
          {hint && <div className="mt-0.5 text-xs text-muted-foreground">{hint}</div>}
        </div>
      </div>
      {/* The row's name is a heading, not a label, so the switch carries its
          own — otherwise a screen reader announces a toggle called nothing. */}
      <Switch checked={on} disabled={busy} onCheckedChange={onChange} aria-label={ariaLabel} />
    </div>
  );
}

/** Empty is not the same as loading, and neither is the same as "you have no
 *  computers at all" — that last one is the only one with somewhere to go. */
function NoComputers() {
  return (
    <div className="px-4 py-6 text-sm">
      <p className="font-medium">No computers connected</p>
      <p className="mt-0.5 text-muted-foreground">
        Connect one and it is offered to every workspace. You can take it out of the ones that do
        not need it here.
      </p>
      <Button
        variant="outline"
        size="sm"
        className="mt-3"
        nativeButton={false}
        render={<Link href="/machines" />}
      >
        Go to Machines
      </Button>
    </div>
  );
}

/** Settings → Workspaces: the computers one workspace may use. */
export function WorkspaceMachines({ workspaceId }: { workspaceId: string }) {
  const machines = useWorkspaceMachines(workspaceId);
  const set = useSetWorkspaceMachine(workspaceId);

  if (machines.isPending) {
    return (
      <div className="space-y-2 px-4 py-3" aria-busy="true">
        <Skeleton className="h-6 w-full" />
        <Skeleton className="h-6 w-2/3" />
      </div>
    );
  }
  if (machines.isError) {
    return (
      <p className="px-4 py-3 text-sm text-muted-foreground">
        {(machines.error as Error).message}
      </p>
    );
  }

  const list: AssignedMachine[] = machines.data ?? [];
  if (list.length === 0) return <NoComputers />;

  const none = list.every((m) => !m.assigned);
  return (
    <>
      {list.map((m) => (
        <Row
          key={m.id}
          label={m.name}
          ariaLabel={`Use ${m.name} in this workspace`}
          hint={m.online ? <Status tone="success" quiet>Online</Status> : "Offline"}
          on={m.assigned}
          busy={set.isPending}
          onChange={(assigned) => set.mutate({ machineId: m.id, assigned })}
        />
      ))}
      {/* Said once, at the bottom, rather than as an error on each row: an
          empty set is a legitimate thing to want — reading old transcripts
          without being able to start anything — but it is not something to
          discover by trying to send. */}
      {none && (
        <p className="border-t px-4 py-3 text-xs text-muted-foreground">
          This workspace has no computer, so nothing can run in it. Its conversations stay readable.
        </p>
      )}
    </>
  );
}

/** A computer's own page: which workspaces offer it. The same row, from the
 *  other end. */
export function MachineWorkspaces({ machineId }: { machineId: string }) {
  const spaces = useMachineWorkspaces(machineId);
  const set = useSetMachineWorkspace(machineId);

  if (spaces.isPending) {
    return (
      <div className="space-y-2" aria-busy="true">
        <Skeleton className="h-6 w-full" />
        <Skeleton className="h-6 w-2/3" />
      </div>
    );
  }
  if (spaces.isError) {
    return (
      <p className="text-sm text-muted-foreground">{(spaces.error as Error).message}</p>
    );
  }

  const list: AssignedWorkspace[] = spaces.data ?? [];
  return (
    <div className={cn("divide-y rounded-lg border")}>
      {list.map((w) => (
        <div key={w.id} className="flex items-center justify-between gap-4 px-4 py-3">
          <span className="min-w-0 truncate text-sm font-medium">{w.name}</span>
          <Switch
            checked={w.assigned}
            disabled={set.isPending}
            onCheckedChange={(assigned) => set.mutate({ workspaceId: w.id, assigned })}
            aria-label={`Offer this computer in ${w.name}`}
          />
        </div>
      ))}
    </div>
  );
}
