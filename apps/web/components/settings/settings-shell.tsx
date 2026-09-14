"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ComponentType, ReactNode } from "react";
import { ChevronRight, Download, KeyRound } from "lucide-react";
import { cn } from "cn";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { ProductSidebar } from "@/components/product-sidebar";

export type SettingsPageId = "password" | "updates";

type Entry = { id: SettingsPageId; label: string; href: string; icon: ComponentType<{ className?: string }> };

// Only what is built. A group earns a place here when it has a page (DESIGN.md §9).
const groups: { label: string; entries: Entry[] }[] = [
  { label: "Account", entries: [{ id: "password", label: "Password", href: "/settings", icon: KeyRound }] },
  { label: "Computers", entries: [{ id: "updates", label: "Updates", href: "/settings/updates", icon: Download }] },
];

export function SettingsShell({ current, children }: { current: SettingsPageId; children: ReactNode }) {
  const router = useRouter();
  const group = groups.find((g) => g.entries.some((e) => e.id === current)) ?? groups[0];
  const entry = group.entries.find((e) => e.id === current) ?? group.entries[0];
  return <SidebarProvider className="h-full"><ProductSidebar current="settings"/><SidebarInset className="min-w-0 rounded-none">
    <div className="flex min-h-0 flex-1 flex-col md:flex-row">
      <aside className="shrink-0 border-b md:flex md:w-60 md:flex-col md:border-b-0 md:border-r">
        <div className="flex h-16 shrink-0 items-center px-5"><h1 className="text-lg font-semibold tracking-tight">Settings</h1></div>
        <div className="px-4 pb-4 md:hidden">
          <label className="sr-only" htmlFor="settings-navigation">Go to a setting</label>
          <select id="settings-navigation" className="h-10 w-full rounded-lg border border-input bg-background px-3 text-sm text-foreground focus-visible:outline-2 focus-visible:outline-ring" value={entry.href} onChange={(event) => router.push(event.target.value)}>
            {groups.map((g) => <optgroup key={g.label} label={g.label}>{g.entries.map((e) => <option key={e.id} value={e.href}>{e.label}</option>)}</optgroup>)}
          </select>
        </div>
        <nav aria-label="Settings" className="hidden min-h-0 space-y-5 overflow-y-auto px-3 pb-6 md:block">
          {groups.map((g) => <section key={g.label} aria-labelledby={`settings-group-${g.label}`}>
            <h2 id={`settings-group-${g.label}`} className="mb-1.5 px-3 text-xs font-medium text-muted-foreground">{g.label}</h2>
            <ul className="space-y-0.5">{g.entries.map((e) => {
              const Icon = e.icon;
              const active = e.id === current;
              return <li key={e.id}><Link href={e.href} aria-current={active ? "page" : undefined} className={cn("flex min-h-9 items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-ring", active ? "bg-accent font-medium text-accent-foreground" : "text-muted-foreground hover:bg-accent hover:text-foreground")}><Icon className="size-4 shrink-0" aria-hidden="true"/><span className="min-w-0 truncate">{e.label}</span></Link></li>;
            })}</ul>
          </section>)}
        </nav>
      </aside>
      <div className="min-w-0 flex-1 overflow-y-auto overscroll-contain">
        <div className="mx-auto w-full max-w-4xl px-4 py-6 sm:px-6 md:px-10 md:py-8">
          <div className="mb-3 flex min-w-0 items-center gap-2 text-xs text-muted-foreground"><span>{group.label}</span><ChevronRight aria-hidden="true" className="size-3 shrink-0"/><span>{entry.label}</span></div>
          {children}
        </div>
      </div>
    </div>
  </SidebarInset></SidebarProvider>;
}
