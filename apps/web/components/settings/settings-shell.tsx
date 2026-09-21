"use client";

import Link from "next/link";
import type { ComponentType, ReactNode } from "react";
import { ChevronRight, Download, KeyRound, Palette, UserRound } from "lucide-react";
import { cn } from "cn";
import { AppHeader, AppShell, PaneHeader } from "@/components/shell/app-shell";

export type SettingsPageId = "account" | "password" | "appearance" | "updates";

type Entry = { id: SettingsPageId; label: string; href: string; icon: ComponentType<{ className?: string }> };

// Only what is built. A group earns a place here when it has a page.
const groups: { label: string; entries: Entry[] }[] = [
  { label: "Account", entries: [
    { id: "account", label: "Account", href: "/settings/account", icon: UserRound },
    { id: "password", label: "Password", href: "/settings/password", icon: KeyRound },
    { id: "appearance", label: "Appearance", href: "/settings/appearance", icon: Palette },
  ] },
  { label: "Computers", entries: [{ id: "updates", label: "Updates", href: "/settings/updates", icon: Download }] },
];

/* Settings in the shell (docs/Decisions.md D-042). The pages that used to be a
   sidebar inside the page are the section's pane; on a phone that pane is the
   screen you land on and a page is the screen below it (D-044), which is why
   `current` can be null — `/settings` is the list there and the Password page
   on a desktop, where showing the list alone would waste the other half. */

export function SettingsShell({ current, children }: { current: SettingsPageId | null; children: ReactNode }) {
  const group = groups.find((g) => g.entries.some((e) => e.id === current));
  const entry = group?.entries.find((e) => e.id === current);

  const pane = <>
    <PaneHeader title="Settings"/>
    <nav aria-label="Settings" className="min-h-0 flex-1 space-y-4 overflow-y-auto p-2 pb-4">
      {groups.map((g) => <section key={g.label} aria-labelledby={`settings-group-${g.label}`}>
        <h3 id={`settings-group-${g.label}`} className="px-2.5 pt-2 pb-1 text-xs font-medium text-muted-foreground">{g.label}</h3>
        <ul className="space-y-0.5">{g.entries.map((e) => {
          const Icon = e.icon;
          const active = e.id === current;
          return <li key={e.id}><Link href={e.href} aria-current={active ? "page" : undefined} className={cn("flex min-h-9 items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-ring", active ? "bg-accent font-medium text-accent-foreground" : "text-muted-foreground hover:bg-accent/55 hover:text-foreground")}><Icon className="size-4 shrink-0" aria-hidden="true"/><span className="min-w-0 truncate">{e.label}</span></Link></li>;
        })}</ul>
      </section>)}
    </nav>
  </>;

  return <AppShell section="settings" pane={pane} detail={current !== null}>
    {entry && group ? <>
      <AppHeader title={entry.label} back={{ href: "/settings", label: "Back to settings" }}/>
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
        <div className="mx-auto w-full max-w-4xl px-4 py-6 sm:px-6 md:px-10 md:py-8">
          {/* "Account > Account" tells nobody anything. A page named after its
              own group shows the group once. */}
          <div className="mb-3 flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
            <span>{group.label}</span>
            {group.label !== entry.label ? <><ChevronRight aria-hidden="true" className="size-3 shrink-0"/><span>{entry.label}</span></> : null}
          </div>
          {children}
        </div>
      </div>
    </> : null}
  </AppShell>;
}
