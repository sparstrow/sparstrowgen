"use client";

import { useState } from "react";
import { useRequestPasswordReset } from "@/lib/queries";
import { AuthLinks, AuthShell, Field, FormError, Lede, SubmitButton, TextLink } from "./shell";

/* Asking for a password reset.
 *
 * The reply is word-for-word the same whether or not the address has an
 * account, because anything else tells a stranger which addresses are
 * registered. Only a real account is actually sent a link.
 */
export function ForgotPassword() {
  const [email, setEmail] = useState("");
  const [sentTo, setSentTo] = useState<string | null>(null);
  const request = useRequestPasswordReset();
  const busy = request.isPending;

  if (sentTo) {
    return (
      <AuthShell title="Check your email">
        <Lede>
          If <span className="font-medium break-all">{sentTo}</span> has a sparstrowgen account, a
          reset link is on its way. It works for 30 minutes.
        </Lede>
        <Lede>Nothing after a few minutes? Check spam, and check it’s the address you registered with.</Lede>
        <AuthLinks>
          <TextLink href="/">Back to sign in</TextLink>
        </AuthLinks>
      </AuthShell>
    );
  }

  return (
    <AuthShell title="Reset your password">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!busy && email.trim()) request.mutate(email, { onSuccess: setSentTo });
        }}
        className="space-y-4"
      >
        <Field
          id="email"
          label="Email"
          type="email"
          value={email}
          onChange={(e) => {
            setEmail(e.target.value);
            if (request.isError) request.reset();
          }}
          disabled={busy}
          autoFocus
          autoComplete="username"
          invalid={request.isError}
          describedBy={request.isError ? "forgot-error" : undefined}
        />

        {request.isError && <FormError id="forgot-error" message={request.error.message} />}

        <SubmitButton busy={busy} busyLabel="Sending" disabled={!email.trim()}>
          Send reset link
        </SubmitButton>
      </form>

      <AuthLinks>
        <TextLink href="/">Back to sign in</TextLink>
      </AuthLinks>
    </AuthShell>
  );
}
