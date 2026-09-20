"use client";

import Link from "next/link";
import { Check, Loader2 } from "lucide-react";
import { cn } from "cn";
import type { Appearance } from "@/lib/api";
import { ACCENTS, MODES, SURFACES } from "@/lib/appearance";
import { useAppearance, useSaveAppearance, useSession } from "@/lib/queries";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { SettingsShell } from "@/components/settings/settings-shell";
import { SettingsCard, SettingsPage, SettingsRow, SettingsSection } from "@/components/settings/settings-rows";

/* Settings → Appearance. Three choices, saved on the account and applied to the
   page the moment they are picked (spec 2026-09-12-appearance-preferences).

   No Save button on purpose: each choice is one click, it is visible
   immediately, and it is reversible by clicking another. When a save fails the
   page says so and goes back to the appearance the account still holds. */

export function AppearanceSurface() {
  const session = useSession();
  const appearance = useAppearance();
  const save = useSaveAppearance();

  if (session.isPending) {
    return <SettingsShell current="appearance">
      <div className="space-y-5"><Skeleton className="h-7 w-40"/><Skeleton className="h-64 w-full"/></div>
    </SettingsShell>;
  }
  if (!session.data?.signedIn) {
    return <SettingsShell current="appearance">
      <div className="max-w-md">
        <h2 className="text-xl font-semibold tracking-tight">Appearance</h2>
        <p className="mt-2 text-sm text-muted-foreground">Sign in to choose how sparstrowgen looks. Your choice is saved to your account.</p>
        <Button className="mt-5" nativeButton={false} render={<Link href="/"/>}>Sign in</Button>
      </div>
    </SettingsShell>;
  }

  return <SettingsShell current="appearance">
    <SettingsPage title="Appearance" description="How sparstrowgen looks. Your choice is saved to your account, so it follows you to every browser you sign in to.">
      {appearance.isPending ? <div className="space-y-3"><Skeleton className="h-24 w-full"/><Skeleton className="h-40 w-full"/></div>
        : appearance.isError ? <SettingsCard><SettingsRow label="Appearance could not be loaded" description={<>{(appearance.error as Error).message} Nothing you have chosen has changed.</>}>
            <Button variant="outline" size="sm" onClick={() => void appearance.refetch()}>Try again</Button>
          </SettingsRow></SettingsCard>
        : <Choices appearance={appearance.data} saving={save.isPending} choose={(next) => save.mutate(next)}/>}
    </SettingsPage>
  </SettingsShell>;
}

function Choices({ appearance, saving, choose }: { appearance: Appearance; saving: boolean; choose: (next: Appearance) => void }) {
  return <>
    <SettingsSection title="Mode" aside={saving ? <span className="flex items-center gap-1.5 text-xs text-muted-foreground"><Loader2 className="size-3.5 motion-safe:animate-spin" aria-hidden="true"/>Saving…</span> : null}>
      <SettingsCard>
        <SettingsRow label="Light or dark" description="System follows this computer's own setting, and changes with it.">
          <div role="radiogroup" aria-label="Mode" className="flex flex-wrap gap-1 rounded-lg border p-1">
            {MODES.map((mode) => <button
              key={mode.id}
              type="button"
              role="radio"
              aria-checked={appearance.mode === mode.id}
              title={mode.description}
              onClick={() => choose({ ...appearance, mode: mode.id })}
              className={cn("rounded-md px-3 py-1.5 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-ring",
                appearance.mode === mode.id ? "bg-card font-medium text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground")}
            >{mode.label}</button>)}
          </div>
        </SettingsRow>
      </SettingsCard>
    </SettingsSection>

    <SettingsSection title="Surface">
      <SettingsCard>
        <SettingsRow label="Background character" description="The neutral the whole app sits on, in both light and dark." align="start">
          <div role="radiogroup" aria-label="Surface" className="grid grid-cols-2 gap-2">
            {SURFACES.map((surface) => {
              const chosen = appearance.surface === surface.id;
              return <button
                key={surface.id}
                type="button"
                role="radio"
                aria-checked={chosen}
                onClick={() => choose({ ...appearance, surface: surface.id })}
                className={cn("flex min-w-36 items-start gap-2 rounded-lg border px-3 py-2 text-left transition-colors focus-visible:outline-2 focus-visible:outline-ring",
                  chosen ? "border-primary bg-card" : "hover:bg-accent")}
              >
                <span className="min-w-0">
                  <span className="flex items-center gap-1.5 text-sm font-medium">
                    {surface.label}
                    {chosen ? <Check className="size-3.5 shrink-0 text-primary" aria-hidden="true"/> : null}
                  </span>
                  <span className="mt-0.5 block text-xs leading-5 text-muted-foreground">{surface.description}</span>
                </span>
              </button>;
            })}
          </div>
        </SettingsRow>
      </SettingsCard>
    </SettingsSection>

    <SettingsSection title="Accent">
      <SettingsCard>
        <SettingsRow label="Emphasis colour" description="Used for buttons and focus. Status, provider and code colours keep their meaning and never change.">
          <div role="radiogroup" aria-label="Accent" className="flex flex-wrap gap-2">
            {ACCENTS.map((accent) => {
              const chosen = appearance.accent === accent.id;
              return <button
                key={accent.id}
                type="button"
                role="radio"
                aria-checked={chosen}
                aria-label={accent.label}
                title={accent.label}
                onClick={() => choose({ ...appearance, accent: accent.id })}
                className={cn("flex size-9 items-center justify-center rounded-full border-2 transition-colors focus-visible:outline-2 focus-visible:outline-ring",
                  chosen ? "border-foreground" : "border-transparent hover:border-border")}
              >
                <span className="flex size-6 items-center justify-center rounded-full" style={{ background: accent.swatch }}>
                  {chosen ? <Check className="size-3.5 text-background" aria-hidden="true"/> : null}
                </span>
              </button>;
            })}
          </div>
        </SettingsRow>
      </SettingsCard>
    </SettingsSection>
  </>;
}
