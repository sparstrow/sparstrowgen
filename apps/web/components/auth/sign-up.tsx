"use client";

import { useState } from "react";
import { useSignUp } from "@/lib/queries";
import { AuthShell, Field, FormError, SubmitButton } from "./shell";

/* Claiming a deployment that nobody has an account on yet.
 *
 * This is the one screen that cannot be an ordinary sign-up form. Behind this
 * login is a program that runs coding agents on the owner's own machine, so a
 * form that anybody who finds the URL can complete is a form that hands a
 * stranger a shell. What proves it is the right person is the setup code the
 * server printed when it started: reading it means having the deployment's
 * logs, which is the same thing as owning the deployment.
 *
 * The code is not a secret worth hiding the EXISTENCE of — saying plainly where
 * it comes from is what makes the screen usable, and somebody who cannot read
 * the log cannot act on the explanation. Being coy here would only confuse the
 * owner, who is the sole person this screen can possibly help.
 *
 * Once this succeeds the server closes sign-up permanently, and this screen is
 * unreachable for the life of the deployment.
 */

/** The server's own floor, repeated here so the form can say it before it
 *  rejects anything. Kept in step with minPassword in
 *  server/internal/api/auth.go — the server is what enforces it. */
const MIN_PASSWORD = 12;

export function SignUp() {
  const [setupCode, setSetupCode] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const signUp = useSignUp();
  const busy = signUp.isPending;

  const tooShort = password.length > 0 && password.length < MIN_PASSWORD;
  const complete =
    setupCode.trim() !== "" && email !== "" && password.length >= MIN_PASSWORD;

  const edit = (set: (v: string) => void) => (v: string) => {
    set(v);
    if (signUp.isError) signUp.reset();
  };

  return (
    <AuthShell
      title="Create your account"
      footer="This account is the only one. Sign-up closes as soon as it exists, and this screen will not come back."
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!busy && complete) signUp.mutate({ setupCode, email, password });
        }}
        className="space-y-4"
      >
        <Field
          id="setup-code"
          label="Setup code"
          hint="Printed in the server's log when it started, on the line that says this app has no account yet. Restarting the server prints a new one."
          value={setupCode}
          onChange={(e) => edit(setSetupCode)(e.target.value)}
          disabled={busy}
          autoFocus
          autoComplete="off"
          spellCheck={false}
          // Monospace because it is a literal string being transcribed by eye
          // from a log, where telling 0 from O is the whole job.
          className="font-mono tracking-wide"
          invalid={signUp.isError}
          describedBy={signUp.isError ? "signup-error" : undefined}
        />

        <Field
          id="email"
          label="Email"
          type="email"
          value={email}
          onChange={(e) => edit(setEmail)(e.target.value)}
          disabled={busy}
          autoComplete="username"
          invalid={signUp.isError}
        />

        <Field
          id="new-password"
          label="Password"
          hint={`At least ${MIN_PASSWORD} characters. Length is what resists guessing, so a phrase you will remember beats a short one with symbols in it.`}
          type="password"
          value={password}
          onChange={(e) => edit(setPassword)(e.target.value)}
          disabled={busy}
          autoComplete="new-password"
          invalid={tooShort || signUp.isError}
        />

        {signUp.isError && (
          <FormError id="signup-error" message={signUp.error.message} />
        )}

        <SubmitButton busy={busy} busyLabel="Creating your account" disabled={!complete}>
          Create account
        </SubmitButton>
      </form>
    </AuthShell>
  );
}
