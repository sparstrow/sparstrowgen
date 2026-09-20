"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, Loader2, MonitorSmartphone, PlugZap, Plus } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import type { Machine } from "@/lib/chat-types";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { Status, type StatusTone } from "@/components/ui/status";
import { Switch } from "@/components/ui/switch";
import { SettingsShell } from "@/components/settings/settings-shell";
import { SettingsCard, SettingsPage, SettingsRow, SettingsSection } from "@/components/settings/settings-rows";

const machineKey = ["machines"] as const;

export function UpdatesSurface() {
  // Waiting, installing and reconnecting happen on the computer and arrive as
  // machines events. The shell holds the one socket for every page now, so
  // this page only has to read the cache those events patch.
  const query = useQuery({ queryKey: machineKey, queryFn: () => api.machines() });
  return <SettingsShell current="updates">
    <SettingsPage title="Updates" description="Control automatic updates for sparstrowgen on each of your computers, or check for a new version manually.">
      {query.isPending ? <UpdatesLoading/>
        : query.isError ? <Empty className="border"><EmptyHeader><EmptyMedia variant="icon"><PlugZap/></EmptyMedia><EmptyTitle>Updates could not be loaded</EmptyTitle><EmptyDescription>{(query.error as Error).message} Nothing on your computers has changed.</EmptyDescription></EmptyHeader><EmptyContent><Button variant="outline" onClick={() => void query.refetch()}>Try again</Button></EmptyContent></Empty>
        : !query.data.length ? <Empty className="border"><EmptyHeader><EmptyMedia variant="icon"><MonitorSmartphone/></EmptyMedia><EmptyTitle>No computers connected</EmptyTitle><EmptyDescription>Pair a computer to see its version and control how it updates.</EmptyDescription></EmptyHeader><EmptyContent><Button nativeButton={false} render={<Link href="/machines"/>}><Plus/>Add computer</Button></EmptyContent></Empty>
        : query.data.map((machine) => <ComputerCard key={machine.id} machine={machine}/>)}
    </SettingsPage>
  </SettingsShell>;
}

function ComputerCard({ machine }: { machine: Machine }) {
  const queryClient = useQueryClient();
  const replace = (next: Machine) => {
    queryClient.setQueryData<Machine[]>(machineKey, (list) => list?.map((m) => (m.id === next.id ? next : m)));
  };
  const check = useMutation({
    mutationFn: () => api.checkForUpdates(machine.id),
    onSuccess: replace,
    onError: (error) => toast.error("Could not check for updates", { description: (error as Error).message }),
  });
  const update = useMutation({
    mutationFn: () => api.applyUpdate(machine.id),
    onSuccess: replace,
    onError: (error) => toast.error("Could not start the update", { description: (error as Error).message }),
  });
  const automatic = useMutation({
    mutationFn: (enabled: boolean) => api.setAutomaticUpdates(machine.id, enabled),
    onSuccess: (next) => { replace(next); toast.success(`Automatic updates ${next.automaticUpdates ? "on" : "off"} for ${next.name}`); },
    onError: (error) => toast.error("Could not save the automatic update setting", { description: (error as Error).message }),
  });

  const status = machine.update;
  // Online but unable to update itself: older than self-updating, or too old.
  const manual = machine.online && !machine.selfUpdates;
  const busy = check.isPending || update.isPending || status.kind === "updating";
  const action = !machine.online ? <Button variant="outline" size="sm" disabled>Check now</Button>
    : manual ? <Button variant="outline" size="sm" nativeButton={false} render={<Link href="/install"/>}><Download/>Download update</Button>
    : check.isPending ? <Button variant="outline" size="sm" disabled><Loader2 className="motion-safe:animate-spin"/>Checking…</Button>
    : update.isPending || status.kind === "updating" ? <Button variant="outline" size="sm" disabled><Loader2 className="motion-safe:animate-spin"/>Updating…</Button>
    : status.kind === "available" ? <Button variant="outline" size="sm" disabled={busy} onClick={() => update.mutate()}>Update now</Button>
    // With automatic updates on, trying again installs; with them off, it only checks.
    : status.kind === "failed" ? <Button variant="outline" size="sm" disabled={busy} onClick={() => (machine.automaticUpdates ? update.mutate() : check.mutate())}>Try again</Button>
    : <Button variant="outline" size="sm" disabled={busy} onClick={() => check.mutate()}>Check now</Button>;

  return <SettingsSection
    title={<span className="flex min-w-0 items-center gap-2"><MonitorSmartphone className="size-4 shrink-0" aria-hidden="true"/><span className="truncate">{machine.name}</span></span>}
    aside={<Status tone={machine.online ? "success" : "neutral"} className="text-xs">{machine.online ? "Online" : "Offline"}</Status>}>
    <SettingsCard>
      <SettingsRow label="Current version" description={machine.tooOld
        ? <Status tone="danger">Too old for sparstrowgen. This computer cannot run agent work until it is updated.</Status>
        : !machine.online && machine.version ? "Last reported before this computer went offline." : undefined}>
        {machine.version === "dev" ? <span className="text-xs text-muted-foreground">Development build</span>
          : machine.version ? <span className="font-mono text-xs text-muted-foreground">v{machine.version}</span>
          : <span className="text-xs text-muted-foreground">Not reported yet</span>}
      </SettingsRow>
      <SettingsRow label="Automatic updates" description={manual ? "Available once this computer has a version that can update itself." : "Checks every hour and installs a new version only when no agent work is running on this computer."}>
        <Switch checked={machine.automaticUpdates} onCheckedChange={(enabled) => automatic.mutate(enabled)} disabled={automatic.isPending || manual} aria-label={`Automatic updates for ${machine.name}`}/>
      </SettingsRow>
      <SettingsRow label="Check for updates" align="start" description={<>
        <p>{manual
          ? "This version cannot update itself. Download the installer and open it on this computer."
          : "Check now regardless of the automatic setting. An update never interrupts agent work: it waits until that work has finished."}</p>
        {!manual && <StatusLine machine={machine}/>}
      </>}>
        {action}
      </SettingsRow>
    </SettingsCard>
  </SettingsSection>;
}

function StatusLine({ machine }: { machine: Machine }) {
  const status = machine.update;
  // A sentence, so only the icon carries the tone; a failure is the exception and reads in full.
  const line = (tone: StatusTone, text: string, quiet = true) =>
    <p role="status" className="mt-2"><Status tone={tone} quiet={quiet}>{text}</Status></p>;
  if (!machine.online) return line("neutral", "This computer is offline. It checks for updates when it reconnects.");
  switch (status.kind) {
    case "current": return line("success", "You're on the latest version.");
    case "available": return line("info", `v${status.version} is available.`);
    case "waiting": return line("pending", `v${status.version} is ready. It installs after the ${status.activeTasks === 1 ? "agent task" : `${status.activeTasks} agent tasks`} running on this computer ${status.activeTasks === 1 ? "finishes" : "finish"}.`);
    case "updating": return line("progress", `Installing v${status.version}. This computer reconnects in a few seconds.`);
    case "failed": return line("danger", status.message, false);
    default: return null;
  }
}

function UpdatesLoading() {
  return <div className="space-y-3" aria-busy="true">
    <Skeleton className="h-5 w-44"/>
    <div className="divide-y rounded-xl border">{[1, 2, 3].map((i) => <div key={i} className="flex min-h-16 items-center justify-between gap-8 px-4 py-3.5"><div className="flex-1 space-y-2"><Skeleton className="h-4 w-36"/><Skeleton className="h-3 w-2/3"/></div><Skeleton className="h-8 w-20"/></div>)}</div>
  </div>;
}
