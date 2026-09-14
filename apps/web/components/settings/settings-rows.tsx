import type { ReactNode } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "cn";

// The settings page vocabulary, after Multica's settings layout: a titled page,
// sections, and cards of label-and-control rows.

export function SettingsPage({ title, description, children }: { title: ReactNode; description?: ReactNode; children: ReactNode }) {
  return <div className="space-y-8">
    <header>
      <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
      {description ? <p className="mt-1 max-w-2xl text-sm leading-6 text-muted-foreground">{description}</p> : null}
    </header>
    {children}
  </div>;
}

export function SettingsSection({ title, aside, children }: { title: ReactNode; aside?: ReactNode; children: ReactNode }) {
  return <section className="space-y-3">
    <div className="flex min-w-0 items-end justify-between gap-4 px-0.5">
      <h3 className="min-w-0 text-sm font-semibold">{title}</h3>
      {aside ? <div className="shrink-0">{aside}</div> : null}
    </div>
    {children}
  </section>;
}

export function SettingsCard({ children, className }: { children: ReactNode; className?: string }) {
  return <Card className={cn("gap-0 py-0 shadow-none", className)}><CardContent className="divide-y px-0">{children}</CardContent></Card>;
}

export function SettingsRow({ label, description, children, align = "center" }: { label: ReactNode; description?: ReactNode; children: ReactNode; align?: "center" | "start" }) {
  return <div className={cn("flex min-h-16 flex-row justify-between gap-4 px-4 py-3.5 sm:gap-8", align === "center" ? "items-center" : "items-start")}>
    <div className="min-w-0 flex-1">
      <div className="text-sm font-medium">{label}</div>
      {description ? <div className="mt-0.5 text-xs leading-5 text-muted-foreground">{description}</div> : null}
    </div>
    <div className="w-auto max-w-[56%] shrink-0">{children}</div>
  </div>;
}
