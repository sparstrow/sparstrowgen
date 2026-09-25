"use client";

import Link from "next/link";
import { ChevronLeft } from "lucide-react";
import { cn } from "cn";
import { useRealtime } from "@/lib/queries";
import { Rail, SectionTray, type Section } from "./rail";
import { LiveStatus } from "./live-status";
import { ResizeHandle } from "./resize-handle";

/* The frame around every screen (docs/Decisions.md D-042, D-043, D-044).

   Wide: an icon rail, the section's own pane beside it, and the page filling
   the rest under one 56px header.

   Narrow: no rail. The pane becomes the screen you land on and the page is the
   screen below it, so there are two levels where the desktop shows both at
   once, and the sections sit in a tray at the bottom. `detail` is which of the
   two levels is showing; it is derived from the route or the selection by
   whoever renders the shell, never stored here. */

export function AppShell({
  section,
  pane,
  detail,
  trayInDetail = true,
  children,
}: {
  section: Section;
  pane: React.ReactNode;
  /** Narrow only: true when the page rather than the list should be showing. */
  detail: boolean;
  /** Chat's detail keeps the composer on the bottom edge, so the tray steps out
   *  of its way there and the back arrow is the way out instead. */
  trayInDetail?: boolean;
  children: React.ReactNode;
}) {
  // The one websocket for the whole app. Every surface is inside this shell, so
  // the live status works on Machines and Settings and not only in Chat.
  useRealtime();

  return (
    <div className="flex h-full flex-col">
      <div className="flex min-h-0 flex-1">
        <Rail current={section} />
        <div
          className={cn(
            // The width is the person's own, dragged on its edge (lib/pane-width.ts).
            "relative min-h-0 w-full shrink-0 flex-col border-r bg-muted/40 md:flex md:w-[var(--pane-list,212px)]",
            detail ? "hidden" : "flex",
          )}
        >
          {pane}
          <ResizeHandle pane="list" edge="right" label="Resize the list" className="hidden md:block" />
        </div>
        <div
          className={cn(
            "min-w-0 flex-1 flex-col md:flex",
            detail ? "flex" : "hidden",
          )}
        >
          {children}
        </div>
      </div>
      {(detail ? trayInDetail : true) && <SectionTray current={section} />}
    </div>
  );
}

/** The pane's own 56px header: the section's name, its one action, and — on a
 *  phone, where this header is the top of the screen — the live status. */
export function PaneHeader({
  title,
  action,
}: {
  title: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex h-14 shrink-0 items-center justify-between gap-2 border-b pr-3 pl-4">
      <h2 className="truncate text-[13px] font-semibold">{title}</h2>
      <div className="flex shrink-0 items-center gap-2">
        <LiveStatus className="text-xs md:hidden" />
        {action}
      </div>
    </div>
  );
}

/** The page's 56px header. One line: what you are looking at on the left, what
 *  it can do and whether it can run on the right. */
export function AppHeader({
  title,
  subtitle,
  back,
  children,
}: {
  title: React.ReactNode;
  /** A second, smaller line under the title — Chat puts the folder here. */
  subtitle?: React.ReactNode;
  /** Where the phone's back arrow goes. A route, or a callback when the level
   *  above is a selection rather than a URL. */
  back: { href: string; label: string } | { onClick: () => void; label: string };
  /** Page controls, before the live status. */
  children?: React.ReactNode;
}) {
  const backClass =
    "-ml-2 inline-flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring md:hidden";

  return (
    <header className="group/header @container/header flex h-14 shrink-0 items-center justify-between gap-3 border-b pr-5 pl-4 md:pl-5">
      <div className="flex min-w-0 items-center gap-2">
        {"href" in back ? (
          <Link href={back.href} aria-label={back.label} className={backClass}>
            <ChevronLeft className="size-5" />
          </Link>
        ) : (
          <button
            type="button"
            onClick={back.onClick}
            aria-label={back.label}
            className={backClass}
          >
            <ChevronLeft className="size-5" />
          </button>
        )}
        <div className="min-w-0">
          <h1 className="truncate text-sm leading-5 font-semibold">{title}</h1>
          {subtitle}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-4">
        {children}
        <LiveStatus className="text-xs" />
      </div>
    </header>
  );
}
