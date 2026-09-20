"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, MonitorSmartphone, PlugZap, Plus, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { api } from "@/lib/api";
import { keys, useMachines } from "@/lib/queries";
import { usePairing, type Pairing } from "@/lib/pairing";
import type { Machine, Provider } from "@/lib/chat-types";
import { useIsMobile } from "@/hooks/use-mobile";
import { Button } from "@/components/ui/button";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { Status, StatusIcon, type StatusTone } from "@/components/ui/status";
import { AppHeader, AppShell, PaneHeader } from "@/components/shell/app-shell";

/* Machines in the shell (docs/Decisions.md D-042): the computers are a pane and
   the one you picked fills the page, rather than a list page that navigates to
   a profile page. On a phone those are the two screens the tray switches
   between (D-044). */

function status(machine: Machine) { return machine.online ? "Online" : "Offline"; }
// Blocked needs a person to act. Waitable resolves itself, so it stays muted like Offline. A value this build does not know is muted too: an older app must not call it a fault.
function providerTone(availability: Provider["availability"]): StatusTone { return availability === "available" ? "success" : availability === "blocked" ? "warning" : "neutral"; }

/* Adding another computer, once one is already connected. The sequence itself
   lives in `usePairing` and is shared with first-run setup (D-046) — this is
   only its wrapping on this screen. */
function PairingPanel({ pair, onConnected }: { pair: Pairing; onConnected: () => void }) {
  if (pair.stage === "approval") return <section className="mt-5 border px-4 py-4"><h2 className="text-sm font-medium">{pair.machineName ? `Approve ${pair.machineName}?` : "Approve this computer?"}</h2><p className="mt-1 text-sm text-muted-foreground">It will be able to run coding agents for your account.</p><div className="mt-4 flex gap-2"><Button size="sm" onClick={() => void pair.approve().then((ok) => { if (ok) { toast.success("Computer connected"); onConnected(); } })} disabled={pair.approving}>{pair.approving ? "Connecting…" : "Approve computer"}</Button><Button size="sm" variant="outline" onClick={() => pair.notNow()} disabled={pair.approving}>Not now</Button></div></section>;
  if (pair.stage === "error") return <section className="mt-5 border px-4 py-4"><div className="flex items-start gap-3"><StatusIcon tone="danger" size="md" className="mt-0.5"/><div><h2 className="text-sm font-medium">This computer could not be added</h2><p className="mt-1 text-sm text-muted-foreground">{pair.error}</p></div></div><div className="mt-4"><Button size="sm" variant="outline" onClick={() => pair.begin()}><RefreshCw/>Try again</Button></div></section>;
  const unanswered = pair.stage === "unanswered";
  return <section className="mt-5 border px-4 py-4"><div className="flex items-start gap-3"><StatusIcon tone={unanswered ? "warning" : "progress"} size="md" className="mt-0.5"/><div><h2 className="text-sm font-medium">{unanswered ? "This computer has not answered yet" : "Looking for this computer"}</h2><p className="mt-1 text-sm text-muted-foreground">{unanswered ? "It may still be starting, or it may not be installed here. Nothing is broken either way — try again, or install it first." : "Approve the browser prompt to open sparstrowgen on this computer."}</p></div></div>{unanswered && <div className="mt-4 flex flex-wrap gap-2"><Button size="sm" variant="outline" onClick={() => pair.retry()}><RefreshCw/>Retry</Button><Button size="sm" variant="outline" nativeButton={false} render={<Link href="/install"/>}><Download/>Install component</Button></div>}</section>;
}

/** The whole section. `id` is the computer the route named; without one the
 *  page shows the first computer on a desktop and nothing on a phone, where the
 *  list is a screen in its own right. */
export function MachinesSurface({ id }: { id?: string }) {
  const isMobile = useIsMobile();
  const list = useMachines();
  const pair = usePairing();
  const running = pair.stage !== "idle";
  const begin = () => pair.begin();

  const machines = list.data ?? [];
  // A desktop shows the list and the profile side by side, so landing on the
  // section with nothing open would waste half the screen on an instruction.
  const activeId = id ?? (isMobile ? undefined : machines[0]?.id);

  const pane = <>
    <PaneHeader title="Machines" action={<Button variant="ghost" size="icon" className="size-7" onClick={begin} disabled={running} aria-label="Add computer"><Plus className="size-4"/></Button>}/>
    <MachinePane list={list.data} pending={list.isPending} failed={list.isError} error={list.error} retry={() => void list.refetch()} begin={begin} activeId={activeId}/>
  </>;

  return <AppShell section="machines" pane={pane} detail={id !== undefined || running}>
    <MachineMain id={activeId} machines={machines} pending={list.isPending} failed={list.isError} begin={begin} pairing={running && <PairingPanel pair={pair} onConnected={() => void list.refetch()}/>}/>
  </AppShell>;
}

function MachinePane({ list, pending, failed, error, retry, begin, activeId }: { list?: Machine[]; pending: boolean; failed: boolean; error: unknown; retry: () => void; begin: () => void; activeId?: string }) {
  if (pending) return <div className="space-y-1 p-2" aria-busy="true" aria-label="Loading computers">{[1, 2].map((i) => <div key={i} className="space-y-1.5 px-2.5 py-2"><Skeleton className="h-3.5 w-3/5"/><Skeleton className="h-3 w-2/5"/></div>)}</div>;
  if (failed) return <div className="flex flex-col items-center gap-2.5 px-4 py-6 text-center text-sm"><Status tone="danger" quiet>Machines could not be loaded</Status><p className="text-xs text-muted-foreground">{(error as Error).message}</p><Button size="sm" variant="outline" onClick={retry}>Try again</Button></div>;
  if (!list?.length) return <div className="flex flex-col items-center gap-2.5 px-4 py-6 text-center"><MonitorSmartphone className="size-6 text-muted-foreground" aria-hidden/><p className="text-sm text-muted-foreground">No computers connected. Pair the computer where your coding agents are installed.</p><Button size="sm" onClick={begin}><Plus className="size-4"/>Add computer</Button></div>;
  return <nav aria-label="Computers" className="min-h-0 flex-1 overflow-y-auto p-2"><ul className="space-y-0.5">{list.map((machine) => <li key={machine.id}><Link href={`/machines/${machine.id}`} aria-current={machine.id === activeId ? "true" : undefined} className={cn("flex items-start gap-2.5 rounded-md px-2.5 py-2 transition-colors focus-visible:outline-2 focus-visible:outline-ring", machine.id === activeId ? "bg-accent" : "hover:bg-accent/55")}><MonitorSmartphone className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden/><span className="min-w-0 flex-1"><span className="block truncate text-sm">{machine.name}</span><span className="mt-0.5 block truncate text-xs text-muted-foreground">{machine.providers.length} provider{machine.providers.length === 1 ? "" : "s"} available</span><Status tone={machine.online ? "success" : "neutral"} size="sm" className="mt-1 text-xs">{status(machine)}</Status></span></Link></li>)}</ul></nav>;
}

function MachineMain({ id, machines, pending, failed, begin, pairing }: { id?: string; machines: Machine[]; pending: boolean; failed: boolean; begin: () => void; pairing: React.ReactNode }) {
  // Pairing owns the page while it runs: it is the one thing on this section
  // that needs watching, and it ends by selecting the computer it connected.
  if (pairing) return <><AppHeader title="Add computer" back={{ href: "/machines", label: "Back to computers" }}/><div className="min-h-0 flex-1 overflow-y-auto"><div className="mx-auto w-full max-w-3xl px-6 py-8">{pairing}</div></div></>;
  if (pending) return <><AppHeader title="Machines" back={{ href: "/machines", label: "Back to computers" }}/><div className="mx-auto w-full max-w-3xl space-y-4 px-6 py-8" aria-busy="true"><Skeleton className="h-8 w-52"/><Skeleton className="h-32 w-full"/></div></>;
  if (failed) return <><AppHeader title="Machines" back={{ href: "/machines", label: "Back to computers" }}/><Empty className="flex-1"><EmptyHeader><EmptyMedia variant="icon"><PlugZap/></EmptyMedia><EmptyTitle>Machines could not be loaded</EmptyTitle><EmptyDescription>Nothing on your computers has changed.</EmptyDescription></EmptyHeader></Empty></>;
  if (!machines.length) return <><AppHeader title="Machines" back={{ href: "/machines", label: "Back to computers" }}/><Empty className="flex-1"><EmptyHeader><EmptyMedia variant="icon"><MonitorSmartphone/></EmptyMedia><EmptyTitle>No computers connected</EmptyTitle><EmptyDescription>Pair the computer where your coding agents are installed. You can add another later.</EmptyDescription></EmptyHeader><EmptyContent><Button onClick={begin}><Plus/>Add computer</Button></EmptyContent></Empty></>;
  const machine = machines.find((m) => m.id === id);
  if (!machine) return <><AppHeader title="Machines" back={{ href: "/machines", label: "Back to computers" }}/><Empty className="flex-1"><EmptyHeader><EmptyMedia variant="icon"><MonitorSmartphone/></EmptyMedia><EmptyTitle>Pick a computer</EmptyTitle><EmptyDescription>Choose one to see the agents it can run.</EmptyDescription></EmptyHeader></Empty></>;
  return <MachineDetail machine={machine}/>;
}

function MachineDetail({ machine }: { machine: Machine }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  // The list carries every field the profile shows, but this keeps the open
  // computer fresh on its own and is what a deep link loads.
  const query = useQuery({ queryKey: ["machines", machine.id], queryFn: () => api.machine(machine.id), initialData: machine });
  const remove = useMutation({ mutationFn: api.disconnectMachine, onSuccess: () => { void queryClient.invalidateQueries({ queryKey: keys.machines }); router.push("/machines"); } });
  const data = query.data;
  return <>
    <AppHeader title={data.name} back={{ href: "/machines", label: "Back to computers" }} subtitle={<span className="block text-xs leading-4 text-muted-foreground">Version {data.version}</span>}>
      <AlertDialog><AlertDialogTrigger render={<Button variant="destructive" size="sm" disabled={remove.isPending}/>}>Disconnect</AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Disconnect this computer?</AlertDialogTitle><AlertDialogDescription>It will no longer be able to run new work until it is paired again.</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel>Cancel</AlertDialogCancel><AlertDialogAction variant="destructive" onClick={() => void remove.mutate(data.id)}>Disconnect</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
    </AppHeader>
    <div className="min-h-0 flex-1 overflow-y-auto"><div className="mx-auto w-full max-w-3xl px-6 py-8">
      <section><h2 className="text-sm font-medium">Agent providers</h2><p className="mt-1 text-sm text-muted-foreground">Providers this computer can make available to new work.</p><div className="mt-4 divide-y border-y">{data.providers.length ? data.providers.map((provider) => <div className="flex items-center justify-between gap-4 py-3" key={provider.id}><span className="font-medium">{provider.label}</span><Status className="text-sm" tone={providerTone(provider.availability)}>{provider.availability === "available" ? "Available" : provider.unavailableReason ?? "Unavailable"}</Status></div>) : <p className="py-5 text-sm text-muted-foreground">Waiting for this computer to connect and report its providers.</p>}</div></section>
      <p className="mt-6 text-xs text-muted-foreground"><Status tone={data.online ? "success" : "neutral"} size="sm">{status(data)}</Status></p>
    </div></div>
  </>;
}

/** The `/machines/[id]` route. Same section, with that computer open. */
export function MachineProfile({ id }: { id: string }) { return <MachinesSurface id={id}/>; }
