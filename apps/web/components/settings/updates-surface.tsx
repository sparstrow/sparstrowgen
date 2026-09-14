"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircle, ArrowDownToLine, Check, Clock, Loader2, MonitorSmartphone, PlugZap, Plus, Wifi, WifiOff } from "lucide-react";
import { toast } from "sonner";
import type { ComputerUpdates, UpdateStatus } from "@/lib/chat-types";
import { cn } from "cn";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { SettingsShell } from "@/components/settings/settings-shell";
import { SettingsCard, SettingsPage, SettingsRow, SettingsSection } from "@/components/settings/settings-rows";
import { checkForUpdates, fetchComputerUpdates, finishUpdate, parseScenario, requestUpdate, setAutomaticUpdates, updateScenarios, type UpdateScenario } from "@/components/settings/updates.mock";

type Key = readonly ["computer-updates", UpdateScenario];

export function UpdatesSurface() {
  const scenario = parseScenario(useSearchParams().get("state"));
  const key: Key = ["computer-updates", scenario];
  const query = useQuery({ queryKey: key, queryFn: () => fetchComputerUpdates(scenario), retry: false });
  return <SettingsShell current="updates">
    <SettingsPage title="Updates" description="Control automatic updates for sparstrowgen on each of your computers, or check for a new version manually.">
      {query.isPending ? <UpdatesLoading/>
        : query.isError ? <Empty className="border"><EmptyHeader><EmptyMedia variant="icon"><PlugZap/></EmptyMedia><EmptyTitle>Updates could not be loaded</EmptyTitle><EmptyDescription>{(query.error as Error).message} Nothing on your computers has changed.</EmptyDescription></EmptyHeader><EmptyContent><Button variant="outline" onClick={() => void query.refetch()}>Try again</Button></EmptyContent></Empty>
        : !query.data.length ? <Empty className="border"><EmptyHeader><EmptyMedia variant="icon"><MonitorSmartphone/></EmptyMedia><EmptyTitle>No computers connected</EmptyTitle><EmptyDescription>Pair a computer to see its version and control how it updates.</EmptyDescription></EmptyHeader><EmptyContent><Button render={<Link href="/machines"/>}><Plus/>Add computer</Button></EmptyContent></Empty>
        : query.data.map((computer) => <ComputerCard key={computer.machineId} computer={computer} queryKey={key}/>)}
    </SettingsPage>
    <MockStates current={scenario}/>
  </SettingsShell>;
}

function ComputerCard({ computer, queryKey }: { computer: ComputerUpdates; queryKey: Key }) {
  const queryClient = useQueryClient();
  const patch = (next: Partial<ComputerUpdates>) => queryClient.setQueryData<ComputerUpdates[]>(queryKey, (list) => list?.map((c) => c.machineId === computer.machineId ? { ...c, ...next } : c));
  const afterStatus = (status: UpdateStatus) => {
    patch({ status });
    if (status.kind === "updating") void finishUpdate().then(patch);
  };
  const check = useMutation({ mutationFn: () => checkForUpdates(computer), onSuccess: afterStatus, onError: (error) => patch({ status: { kind: "failed", message: `Could not check for updates: ${(error as Error).message}` } }) });
  const update = useMutation({ mutationFn: () => requestUpdate(computer), onSuccess: afterStatus });
  const automatic = useMutation({
    mutationFn: setAutomaticUpdates,
    onSuccess: (enabled) => { patch({ automatic: enabled }); toast.success(`Automatic updates ${enabled ? "on" : "off"} for ${computer.name}`); },
    onError: () => toast.error("Could not save the automatic update setting"),
  });

  const { status } = computer;
  const busy = check.isPending || update.isPending || status.kind === "updating";
  const action = computer.tooOld
    ? <Button variant="outline" size="sm" render={<Link href="/install"/>}>Download update</Button>
    : check.isPending ? <Button variant="outline" size="sm" disabled><Loader2 className="motion-safe:animate-spin"/>Checking…</Button>
    : status.kind === "updating" ? <Button variant="outline" size="sm" disabled><Loader2 className="motion-safe:animate-spin"/>Updating…</Button>
    : status.kind === "available" ? <Button variant="outline" size="sm" disabled={busy || !computer.online} onClick={() => update.mutate()}>Update now</Button>
    : status.kind === "failed" ? <Button variant="outline" size="sm" disabled={busy || !computer.online} onClick={() => check.mutate()}>Try again</Button>
    : <Button variant="outline" size="sm" disabled={busy || !computer.online} onClick={() => check.mutate()}>Check now</Button>;

  return <SettingsSection
    title={<span className="flex min-w-0 items-center gap-2"><MonitorSmartphone className="size-4 shrink-0" aria-hidden="true"/><span className="truncate">{computer.name}</span></span>}
    aside={<span className="flex items-center gap-1.5 text-xs text-muted-foreground">{computer.online ? <Wifi className="size-3.5" aria-hidden="true"/> : <WifiOff className="size-3.5" aria-hidden="true"/>}{computer.online ? "Online" : "Offline"}</span>}>
    <SettingsCard>
      <SettingsRow label="Current version" description={computer.tooOld
        ? <span className="inline-flex items-center gap-1.5 text-destructive"><AlertCircle className="size-3.5 shrink-0" aria-hidden="true"/>Too old for sparstrowgen. This computer cannot run agent work until it is updated.</span>
        : computer.online ? undefined : "Last reported before this computer went offline."}>
        <span className="font-mono text-xs text-muted-foreground">v{computer.version}</span>
      </SettingsRow>
      <SettingsRow label="Automatic updates" description={computer.tooOld ? "Available once this computer has a version that can update itself." : "Checks every hour and installs a new version only when no agent work is running on this computer."}>
        <Switch checked={computer.automatic && !computer.tooOld} onCheckedChange={(enabled) => automatic.mutate(enabled)} disabled={automatic.isPending || computer.tooOld} aria-label={`Automatic updates for ${computer.name}`}/>
      </SettingsRow>
      <SettingsRow label="Check for updates" align="start" description={<>
        <p>{computer.tooOld
          ? "This version cannot update itself. Download the installer and open it on this computer."
          : "Check now regardless of the automatic setting. An update never interrupts agent work: it waits until that work has finished."}</p>
        {!computer.tooOld && <StatusLine computer={computer}/>}
      </>}>
        {action}
      </SettingsRow>
    </SettingsCard>
  </SettingsSection>;
}

function StatusLine({ computer }: { computer: ComputerUpdates }) {
  const { status } = computer;
  const line = (icon: React.ReactNode, text: string, tone?: "destructive") =>
    <p role="status" className={cn("mt-2 inline-flex items-start gap-1.5", tone === "destructive" ? "text-destructive" : "text-foreground")}>{icon}<span>{text}</span></p>;
  if (!computer.online) return line(<WifiOff className="mt-0.5 size-3.5 shrink-0" aria-hidden="true"/>, "This computer is offline. It checks for updates when it reconnects.");
  switch (status.kind) {
    case "current": return line(<Check className="mt-0.5 size-3.5 shrink-0" aria-hidden="true"/>, "You're on the latest version.");
    case "available": return line(<ArrowDownToLine className="mt-0.5 size-3.5 shrink-0" aria-hidden="true"/>, `v${status.version} is available.`);
    case "waiting": return line(<Clock className="mt-0.5 size-3.5 shrink-0" aria-hidden="true"/>, `v${status.version} is ready. It installs after the ${status.activeTasks === 1 ? "agent task" : `${status.activeTasks} agent tasks`} running on this computer ${status.activeTasks === 1 ? "finishes" : "finish"}.`);
    case "updating": return line(<Loader2 className="mt-0.5 size-3.5 shrink-0 motion-safe:animate-spin" aria-hidden="true"/>, `Installing v${status.version}. This computer reconnects in a few seconds.`);
    case "failed": return line(<AlertCircle className="mt-0.5 size-3.5 shrink-0" aria-hidden="true"/>, status.message, "destructive");
    case "unchecked": return null;
    default: return null;
  }
}

function UpdatesLoading() {
  return <div className="space-y-3" aria-busy="true">
    <Skeleton className="h-5 w-44"/>
    <div className="divide-y rounded-xl border">{[1, 2, 3].map((i) => <div key={i} className="flex min-h-16 items-center justify-between gap-8 px-4 py-3.5"><div className="flex-1 space-y-2"><Skeleton className="h-4 w-36"/><Skeleton className="h-3 w-2/3"/></div><Skeleton className="h-8 w-20"/></div>)}</div>
  </div>;
}

// Review scaffolding for mock data only. Removed with updates.mock.ts.
function MockStates({ current }: { current: UpdateScenario }) {
  return <div className="mx-auto mt-10 w-full max-w-4xl px-4 pb-8 sm:px-6 md:px-10">
    <div className="rounded-lg border border-dashed px-4 py-3 text-xs text-muted-foreground">
      <p className="font-medium text-foreground">Preview with mock data — not part of the product</p>
      <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1">{updateScenarios.map((s) => <Link key={s} href={`/settings/updates?state=${s}`} aria-current={s === current ? "page" : undefined} className={cn("underline-offset-4 hover:underline", s === current && "font-medium text-foreground underline")}>{s}</Link>)}</div>
    </div>
  </div>;
}
