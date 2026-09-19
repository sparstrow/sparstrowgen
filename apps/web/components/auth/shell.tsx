"use client";

import Link from "next/link";
import { Loader2 } from "lucide-react";
import { cn } from "cn";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Status } from "@/components/ui/status";

/* The furniture shared by every door into the app: sign in, create an account,
 * the pages an email link opens, and password reset.
 *
 * It is one module rather than several similar screens because they are the
 * same screen to look at, and the failure mode of copying it is that one of
 * them quietly stops matching the others — a different field height, an error
 * in a different place. Nobody sees them all in the same session, so nobody
 * notices.
 *
 * Nothing here decides anything. What each door asks for, and what happens when
 * it opens, belongs to the screen that uses it.
 */

/** The row of ways onward under a form — sign in instead, forgot password,
 *  try again. Spread to both edges so two of them never read as one sentence. */
export function AuthLinks({ children }: { children: React.ReactNode }) {
  return <div className="mt-5 flex flex-wrap justify-between gap-3 text-sm">{children}</div>;
}

const quiet =
  "text-muted-foreground underline-offset-4 outline-none hover:text-foreground hover:underline focus-visible:text-foreground focus-visible:underline disabled:pointer-events-none disabled:opacity-60";

/** A way onward that goes to another page. Muted, because it is never the
 *  thing the screen is for. */
export function TextLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className={quiet}>
      {children}
    </Link>
  );
}

/** The same, for a way onward that stays on this page. */
export function TextButton({
  onClick,
  disabled,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button type="button" onClick={onClick} disabled={disabled} className={quiet}>
      {children}
    </button>
  );
}

/** A sentence the screen needs you to read before doing anything. */
export function Lede({ children }: { children: React.ReactNode }) {
  return <p className="mb-4 text-sm leading-relaxed">{children}</p>;
}

/** A value the person cannot change here, labelled like a field so it lines
 *  up with the fields around it — but not a disabled input, which reads as
 *  something temporarily unavailable. */
export function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-1">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="text-sm break-all">{value}</p>
    </div>
  );
}

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
    <p id={id} role="alert" className="text-sm">
      <Status tone="danger">{message}</Status>
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
