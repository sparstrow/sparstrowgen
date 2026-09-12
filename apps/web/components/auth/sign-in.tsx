"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useSignIn } from "@/lib/queries";
import { AuthShell, Field, FormError, SubmitButton } from "./shell";

/* The way in, for somebody who already has an account here.
 *
 * No "forgot password" link, because there is nobody to email: this deployment
 * has one account and no mail server, and a link that could not work would be
 * worse than not offering one. What actually recovers a lost password is
 * documented where a locked-out owner can still read it
 * (docs/runbooks/deploy.md), not behind the login he cannot get through.
 *
 * The wording never distinguishes a wrong password from an email with no
 * account — the server sends one message for both, and telling them apart tells
 * a stranger which addresses exist here.
 */
export function SignIn() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const signIn = useSignIn();
  const busy = signIn.isPending;

  // Clear a previous failure the moment he starts correcting it. Leaving "that
  // did not match an account" under a field he is actively retyping reads as
  // though the new attempt failed too.
  const edit = (set: (v: string) => void) => (v: string) => {
    set(v);
    if (signIn.isError) signIn.reset();
  };

  return (
    <AuthShell
      title="Sign in"
      footer="This signs you in to the machine that runs your agents. Sessions last a week, and signing out ends them everywhere."
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!busy && email && password) signIn.mutate({ email, password });
        }}
        className="space-y-4"
      >
        <Field
          id="email"
          label="Email"
          type="email"
          value={email}
          onChange={(e) => edit(setEmail)(e.target.value)}
          disabled={busy}
          autoFocus
          autoComplete="username"
          invalid={signIn.isError}
          describedBy={signIn.isError ? "signin-error" : undefined}
        />

        <Field
          id="password"
          label="Password"
          type="password"
          value={password}
          onChange={(e) => edit(setPassword)(e.target.value)}
          disabled={busy}
          autoComplete="current-password"
          invalid={signIn.isError}
          describedBy={signIn.isError ? "signin-error" : undefined}
        />

        {signIn.isError && (
          <FormError id="signin-error" message={signIn.error.message} />
        )}

        <SubmitButton busy={busy} busyLabel="Signing in" disabled={!email || !password}>
          Sign in
        </SubmitButton>
      </form>
    </AuthShell>
  );
}

/** Shown while the browser is asking what it is allowed to see. Almost always a
 *  single frame — but on a cold server it is the difference between a blank
 *  page and a page that is visibly doing something. */
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
export function SignInUnreachable({
  error,
  onRetry,
}: {
  error: string;
  onRetry: () => void;
}) {
  return (
    <main className="flex min-h-dvh items-center justify-center px-6">
      <div className="w-full max-w-sm space-y-4">
        <h1 className="text-lg font-medium tracking-tight">
          Can&rsquo;t reach the server
        </h1>
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
