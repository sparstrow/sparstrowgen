"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Loader2, LockKeyhole } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

/* The way in.
 *
 * One field, because there is one user. No "forgot password", because there is
 * nobody to email and no account to recover — the password is set in the
 * server's own configuration, and the recovery procedure is to change it there
 * (docs/runbooks/deploy.md). Offering a link that could not work would be worse
 * than not offering one.
 *
 * The wording never says whether a password EXISTS, only whether this one was
 * right, and the server's throttling message is passed through as written: "try
 * again in 8s" is the one thing the owner actually needs when he has mistyped
 * four times, and it tells an attacker nothing they could not measure.
 */
export function SignIn({ onSignedIn }: { onSignedIn: () => void }) {
  const [password, setPassword] = useState("");

  const signIn = useMutation({
    mutationFn: () => api.signIn(password),
    onSuccess: () => {
      setPassword("");
      onSignedIn();
    },
  });

  const busy = signIn.isPending;

  return (
    <main className="flex min-h-dvh items-center justify-center px-6 py-12">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex items-center gap-2.5">
          <LockKeyhole className="size-5 text-muted-foreground" aria-hidden />
          <h1 className="text-lg font-medium tracking-tight">sparstrowgen</h1>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (!busy && password) signIn.mutate();
          }}
          className="space-y-4"
        >
          <div className="space-y-2">
            <label htmlFor="password" className="block text-sm text-muted-foreground">
              Password
            </label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                // Clear a previous failure the moment he starts correcting it.
                // Leaving "that password is not right" under a field he is
                // actively retyping reads as though the new attempt failed too.
                if (signIn.isError) signIn.reset();
              }}
              disabled={busy}
              autoFocus
              autoComplete="current-password"
              aria-invalid={signIn.isError || undefined}
              aria-describedby={signIn.isError ? "signin-error" : undefined}
              className="text-base"
            />
          </div>

          {signIn.isError && (
            <p id="signin-error" role="alert" className="text-sm text-destructive">
              {signIn.error.message}
            </p>
          )}

          <Button type="submit" disabled={busy || !password} className="w-full">
            {busy ? (
              <>
                <Loader2 className="size-4 animate-spin" aria-hidden />
                Signing in
              </>
            ) : (
              "Sign in"
            )}
          </Button>
        </form>

        <p className="mt-8 text-xs leading-relaxed text-muted-foreground">
          This signs you in to the machine that runs your agents. Sessions last a
          week, and signing out ends them everywhere.
        </p>
      </div>
    </main>
  );
}

/** Shown while the browser is asking whether it already has a session. Almost
 *  always a single frame — but on a cold server it is the difference between a
 *  blank page and a page that is visibly doing something. */
export function SignInChecking() {
  return (
    <main className="flex min-h-dvh items-center justify-center">
      <Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />
      <span className="sr-only">Checking whether you are signed in</span>
    </main>
  );
}

/** The server could not be reached at all, which is a different thing from
 *  being signed out and must not be shown as a login failure — no password will
 *  fix it, and saying "wrong password" would send the owner hunting for the
 *  wrong problem. */
export function SignInUnreachable({ error, onRetry }: { error: string; onRetry: () => void }) {
  return (
    <main className="flex min-h-dvh items-center justify-center px-6">
      <div className="w-full max-w-sm space-y-4">
        <h1 className="text-lg font-medium tracking-tight">Can&rsquo;t reach the server</h1>
        <p className="text-sm text-muted-foreground">
          The app is running, but the API did not answer. It may be starting up,
          or stopped.
        </p>
        <p className="rounded-md border bg-muted/40 px-3 py-2 font-mono text-xs text-muted-foreground">
          {error}
        </p>
        <Button variant="secondary" onClick={onRetry} className="w-full">
          Try again
        </Button>
      </div>
    </main>
  );
}
