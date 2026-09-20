"use client";

import Link from "next/link";
import { useEffect } from "react";
import {
  MessageSquareText,
  MonitorSmartphone,
  PanelLeft,
  Settings,
  type LucideIcon,
} from "lucide-react";
import { cn } from "cn";
import { AccountMenu } from "@/components/auth/account-menu";
import { useShellView } from "@/lib/store";

/* The primary navigation, in two shapes for two shapes of screen.

   On a desktop it is a 60px rail of icons that widens to 224px over the top of
   the page on hover or keyboard focus, so the layout never shifts as the mouse
   crosses it. Pinning it (D-043) keeps it open and then it does take its own
   column, because a rail that is always there should not be covering anything.
   The pin earns its place as the destinations grow past three — a column of
   unlabelled glyphs is a memory test — and it is what makes the rail usable on
   a touch screen, where there is no hover at all.

   Below 768px there is no rail: the same three entries become the bottom tray
   (D-044), which is where a thumb already is. */

export type Section = "chat" | "machines" | "settings";

const NAV: { id: Section; label: string; href: string; icon: LucideIcon }[] = [
  { id: "chat", label: "Chat", href: "/", icon: MessageSquareText },
  { id: "machines", label: "Machines", href: "/machines", icon: MonitorSmartphone },
  { id: "settings", label: "Settings", href: "/settings", icon: Settings },
];

export function Rail({ current }: { current: Section }) {
  const pinned = useShellView((s) => s.railPinned);
  const togglePin = useShellView((s) => s.toggleRailPin);
  const loadPin = useShellView((s) => s.loadRailPin);

  // The server has no way to know, so the first paint is unpinned and the
  // browser corrects it. Doing this in an effect rather than in the store's
  // initial state keeps the server and client markup identical.
  useEffect(loadPin, [loadPin]);

  return (
    <div
      className={cn(
        "relative z-30 hidden shrink-0 md:block",
        pinned ? "w-56" : "w-[60px]",
      )}
    >
      <aside
        aria-label="Primary"
        className={cn(
          "group/rail absolute inset-y-0 left-0 flex flex-col overflow-hidden border-r bg-background",
          "transition-[width] duration-200 motion-reduce:transition-none",
          pinned ? "w-56" : "w-[60px] hover:w-56 focus-within:w-56",
        )}
      >
        <div className="flex h-14 shrink-0 items-center justify-between gap-3 border-b pr-3 pl-[22px]">
          <span className="flex items-center gap-3 text-sm font-semibold tracking-tight whitespace-nowrap">
            {/* No logo: the design system says the wordmark is plain type, so
                the collapsed rail shows its first letter rather than inventing
                a mark. */}
            <span aria-hidden="true">s</span>
            <Label pinned={pinned}>sparstrowgen</Label>
          </span>
          <button
            type="button"
            onClick={togglePin}
            aria-pressed={pinned}
            aria-label={pinned ? "Let the menu collapse" : "Keep the menu open"}
            className={cn(
              "inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground",
              "transition-opacity duration-150 motion-reduce:transition-none hover:bg-accent hover:text-foreground",
              "focus-visible:opacity-100 focus-visible:outline-2 focus-visible:outline-ring",
              pinned
                ? "bg-accent text-foreground opacity-100"
                : "opacity-0 group-hover/rail:opacity-100 group-focus-within/rail:opacity-100",
            )}
          >
            <PanelLeft className="size-4" />
          </button>
        </div>

        <nav aria-label="Sections" className="flex flex-1 flex-col gap-0.5 p-2">
          {NAV.map((n) => {
            const Icon = n.icon;
            const active = n.id === current;
            return (
              <Link
                key={n.id}
                href={n.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex items-center gap-3 rounded-md px-4 py-2.5 text-sm font-medium whitespace-nowrap",
                  "focus-visible:outline-2 focus-visible:outline-ring",
                  active
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:bg-accent/60",
                )}
              >
                <Icon className="size-4 shrink-0" aria-hidden="true" />
                <Label pinned={pinned}>{n.label}</Label>
              </Link>
            );
          })}
        </nav>

        <div className="flex shrink-0 items-center gap-2 border-t px-3 py-2.5">
          <AccountMenu />
          <Label pinned={pinned}>
            <span className="text-xs text-muted-foreground">Account</span>
          </Label>
        </div>
      </aside>
    </div>
  );
}

/** Hidden until the rail is open. Opacity rather than display, so the labels
 *  are in the accessibility tree and a screen reader reads a named link even
 *  while the rail looks like icons. */
function Label({ pinned, children }: { pinned: boolean; children: React.ReactNode }) {
  return (
    <span
      className={cn(
        "truncate transition-opacity duration-150 motion-reduce:transition-none",
        pinned
          ? "opacity-100"
          : "opacity-0 group-hover/rail:opacity-100 group-focus-within/rail:opacity-100",
      )}
    >
      {children}
    </span>
  );
}

/** The phone's primary navigation. Same three entries, same order. */
export function SectionTray({ current }: { current: Section }) {
  return (
    <nav
      aria-label="Sections"
      className="flex shrink-0 border-t bg-background md:hidden"
    >
      {NAV.map((n) => {
        const Icon = n.icon;
        const active = n.id === current;
        return (
          <Link
            key={n.id}
            href={n.href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "flex flex-1 flex-col items-center justify-center gap-0.5 py-2 pb-2.5 text-[11px] leading-3.5 font-medium",
              "focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring",
              active ? "text-foreground" : "text-muted-foreground",
            )}
          >
            <Icon className="size-5" aria-hidden="true" />
            {n.label}
          </Link>
        );
      })}
    </nav>
  );
}
