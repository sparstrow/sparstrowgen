"use client";

import { useState } from "react";
import Link from "next/link";
import { ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { useChangePassword, useSession } from "@/lib/queries";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { SettingsShell } from "@/components/settings/settings-shell";
import { SettingsPage } from "@/components/settings/settings-rows";

export function SettingsSurface() {
  const session = useSession();
  const change = useChangePassword();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirmation, setConfirmation] = useState("");

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (next !== confirmation) {
      toast.error("New passwords do not match");
      return;
    }
    try {
      await change.mutateAsync({ current, next });
      setCurrent(""); setNext(""); setConfirmation("");
      toast.success("Password changed", { description: "Your other signed-in sessions were ended." });
    } catch (error) {
      toast.error("Password was not changed", { description: (error as Error).message });
    }
  }

  return <SettingsShell current="password">
    {session.isPending ? <div className="space-y-5"><Skeleton className="h-7 w-36"/><Skeleton className="h-52 w-full"/></div> : !session.data?.signedIn ? <div className="max-w-md"><h2 className="text-xl font-semibold tracking-tight">Password</h2><p className="mt-2 text-sm text-muted-foreground">Sign in to manage your password.</p><Button className="mt-5" nativeButton={false} render={<Link href="/"/>}>Sign in</Button></div> : <SettingsPage title="Password" description="Changing your password signs out your other devices. Use at least 12 characters.">
      <form className="max-w-md space-y-4" onSubmit={(event)=>void submit(event)}><label className="grid gap-1.5 text-sm font-medium">Current password<Input type="password" autoComplete="current-password" value={current} onChange={(e)=>setCurrent(e.target.value)} required /></label><label className="grid gap-1.5 text-sm font-medium">New password<Input type="password" autoComplete="new-password" minLength={12} value={next} onChange={(e)=>setNext(e.target.value)} required /></label><label className="grid gap-1.5 text-sm font-medium">Confirm new password<Input type="password" autoComplete="new-password" minLength={12} value={confirmation} onChange={(e)=>setConfirmation(e.target.value)} required /></label><Button type="submit" disabled={change.isPending}>{change.isPending ? "Changing password…" : "Change password"}</Button></form>
      <div className="flex items-start gap-2 border-t pt-5 text-sm text-muted-foreground"><ShieldCheck className="mt-0.5 size-4 shrink-0"/><p>Signed in as {session.data?.email ?? "your account"}. Your password is never shown or stored in the browser.</p></div>
    </SettingsPage>}
  </SettingsShell>;
}
