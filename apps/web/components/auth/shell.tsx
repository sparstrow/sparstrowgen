"use client";

import { Loader2 } from "lucide-react";
import { cn } from "cn";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

/* The furniture shared by the two doors into the app.
 *
 * It is one module rather than two similar screens because they are the same
 * screen to look at, and the failure mode of copying it is that one of them
 * quietly stops matching the other — a different field height, an error in a
 * different place. Nobody sees both in the same session, so nobody notices.
 *
 * Nothing here decides anything. What the door asks for, and what happens when
 * it opens, belongs to sign-in.tsx and sign-up.tsx.
 */

export function AuthShell({
  title,
  children,
  footer,
}: {
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <main className="flex min-h-dvh items-center justify-center px-6 py-12">
      <div className="w-full max-w-sm">
        <h1 className="mb-1 text-lg font-medium tracking-tight">sparstrowgen</h1>
        <p className="mb-8 text-sm text-muted-foreground">{title}</p>
        {children}
        {footer && (
          <div className="mt-8 text-xs leading-relaxed text-muted-foreground">
            {footer}
          </div>
        )}
      </div>
    </main>
  );
}

/** A labelled field.
 *
 *  `hint` sits under the input rather than above it: it is read when somebody
 *  has stopped and is unsure, and putting it between the label and the box
 *  pushes the box away from the thing that names it. */
export function Field({
  id,
  label,
  hint,
  invalid,
  describedBy,
  className,
  ...input
}: {
  id: string;
  label: string;
  hint?: React.ReactNode;
  invalid?: boolean;
  describedBy?: string;
} & React.ComponentProps<typeof Input>) {
  const hintId = hint ? `${id}-hint` : undefined;
  const described = [describedBy, hintId].filter(Boolean).join(" ") || undefined;

  return (
    <div className="space-y-2">
      <label htmlFor={id} className="block text-sm text-muted-foreground">
        {label}
      </label>
      <Input
        id={id}
        aria-invalid={invalid || undefined}
        aria-describedby={described}
        // Merged, not overwritten: a caller passing its own className would
        // otherwise silently drop the 16px text that stops iOS zooming the page
        // on focus.
        className={cn("text-base", className)}
        {...input}
      />
      {hint && (
        <p id={hintId} className="text-xs leading-relaxed text-muted-foreground">
          {hint}
        </p>
      )}
    </div>
  );
}

/** What went wrong, in the server's own words.
 *
 *  The server writes these to be read by the owner — "try again in 8s" is the
 *  one thing he actually needs after mistyping four times — so they are
 *  surfaced rather than replaced with something generic. */
export function FormError({ id, message }: { id: string; message: string }) {
  return (
    <p id={id} role="alert" className="text-sm text-destructive">
      {message}
    </p>
  );
}

export function SubmitButton({
  busy,
  busyLabel,
  children,
  disabled,
}: {
  busy: boolean;
  busyLabel: string;
  children: React.ReactNode;
  disabled?: boolean;
}) {
  return (
    <Button type="submit" disabled={busy || disabled} className="w-full">
      {busy ? (
        <>
          <Loader2 className="size-4 animate-spin" aria-hidden />
          {busyLabel}
        </>
      ) : (
        children
      )}
    </Button>
  );
}
