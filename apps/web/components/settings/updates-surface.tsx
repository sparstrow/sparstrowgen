"use client";

import { useCallback } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircle, ArrowDownToLine, Check, Clock, Download, Loader2, MonitorSmartphone, PlugZap, Plus, Wifi, WifiOff } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { api } from "@/lib/api";
import type { Machine } from "@/lib/chat-types";
import { useRealtime } from "@/lib/queries";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { SettingsShell } from "@/components/settings/settings-shell";
import { SettingsCard, SettingsPage, SettingsRow, SettingsSection } from "@/components/settings/settings-rows";

const machineKey = ["machines"] as const;

export function UpdatesSurface() {
  // Waiting, installing and reconnecting happen on the computer and arrive as
  // machines events, so this page listens like Chat does.
  const ignoreDaemon = useCallback(() => {}, []);
  useRealtime(ignoreDaemon);
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
    aside={<span className="flex items-center gap-1.5 text-xs text-muted-foreground">{machine.online ? <Wifi className="size-3.5 text-success" aria-hidden="true"/> : <WifiOff className="size-3.5" aria-hidden="true"/>}<span className={machine.online ? "text-success-text" : undefined}>{machine.online ? "Online" : "Offline"}</span></span>}>
    <SettingsCard>
      <SettingsRow label="Current version" description={machine.tooOld
        ? <span className="inline-flex items-center gap-1.5 text-destructive"><AlertCircle className="size-3.5 shrink-0" aria-hidden="true"/>Too old for sparstrowgen. This computer cannot run agent work until it is updated.</span>
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
  const line = (icon: React.ReactNode, text: string, tone?: "destructive") =>
    <p role="status" className={cn("mt-2 inline-flex items-start gap-1.5", tone === "destructive" ? "text-destructive" : "text-foreground")}>{icon}<span>{text}</span></p>;
  if (!machine.online) return line(<WifiOff className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden="true"/>, "This computer is offline. It checks for updates when it reconnects.");
  switch (status.kind) {
    case "current": return line(<Check className="mt-0.5 size-3.5 shrink-0 text-success" aria-hidden="true"/>, "You're on the latest version.");
    case "available": return line(<ArrowDownToLine className="mt-0.5 size-3.5 shrink-0 text-info" aria-hidden="true"/>, `v${status.version} is available.`);
    case "waiting": return line(<Clock className="mt-0.5 size-3.5 shrink-0 text-warning" aria-hidden="true"/>, `v${status.version} is ready. It installs after the ${status.activeTasks === 1 ? "agent task" : `${status.activeTasks} agent tasks`} running on this computer ${status.activeTasks === 1 ? "finishes" : "finish"}.`);
    case "updating": return line(<Loader2 className="mt-0.5 size-3.5 shrink-0 text-info motion-safe:animate-spin" aria-hidden="true"/>, `Installing v${status.version}. This computer reconnects in a few seconds.`);
    case "failed": return line(<AlertCircle className="mt-0.5 size-3.5 shrink-0" aria-hidden="true"/>, status.message, "destructive");
    default: return null;
  }
}

function UpdatesLoading() {
  return <div className="space-y-3" aria-busy="true">
    <Skeleton className="h-5 w-44"/>
    <div className="divide-y rounded-xl border">{[1, 2, 3].map((i) => <div key={i} className="flex min-h-16 items-center justify-between gap-8 px-4 py-3.5"><div className="flex-1 space-y-2"><Skeleton className="h-4 w-36"/><Skeleton className="h-3 w-2/3"/></div><Skeleton className="h-8 w-20"/></div>)}</div>
  </div>;
}
