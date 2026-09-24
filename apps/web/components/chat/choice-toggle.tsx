"use client";

import { Button } from "@/components/ui/button";

/** Two or three buttons, one pressed, rather than one that flips: a single
 *  toggle never says what it would switch to, and these are reached for
 *  precisely when you already distrust what is on screen.
 *
 *  The hand-built control the design system lists as "not in this system yet".
 *  This is the one copy of it, used by the Rendered / Raw switch and by the raw
 *  exchange's own switches, so adding it to the system is one move. */
export function ChoiceToggle<T extends string>({
  options,
  value,
  onChange,
  label,
  className,
}: {
  options: { id: T; label: string }[];
  value: T;
  onChange: (v: T) => void;
  /** What the group chooses between, for a screen reader. */
  label: string;
  className?: string;
}) {
  return (
    <div
      className={`flex shrink-0 items-center gap-0.5 rounded-lg bg-muted p-0.5 ${className ?? ""}`}
      role="group"
      aria-label={label}
    >
      {options.map((o) => (
        <Button
          key={o.id}
          variant="ghost"
          size="xs"
          // Selection is a surface lifted out of the track, not a tint: in the
          // light theme `secondary` and the ghost hover resolve to the same
          // value, so a tinted selection is invisible the moment the cursor is
          // over either half.
          className={
            value === o.id
              ? "bg-background text-foreground shadow-sm hover:bg-background"
              : "text-muted-foreground"
          }
          aria-pressed={value === o.id}
          onClick={() => onChange(o.id)}
        >
          {o.label}
        </Button>
      ))}
    </div>
  );
}
