"use client";

import { useEffect, useState } from "react";
import type { RegisterResult } from "@/lib/api";
import { useRegister, useResendConfirmation } from "@/lib/queries";
import {
  AuthLinks,
  AuthShell,
  Field,
  FormError,
  Lede,
  SubmitButton,
  TextButton,
  TextLink,
} from "./shell";

/* Creating an account, for anybody who has the address.
 *
 * Only an email is asked for. The password comes after the address is proved,
 * so an account nobody has confirmed never exists (design-system/DECISIONS.md
 * DD-002).
 *
 * The server decides what happens next and the screen only reports it: an
 * invited address gets "check your email", anything else becomes a request to
 * the owner (DD-001). An address that already has an account gets the same
 * "check your email" as a new one — the email itself says it already exists —
 * so this form never tells a stranger which addresses are registered.
 */

const RESEND_GAP_SECONDS = 30;

export function Register() {
  const [email, setEmail] = useState("");
  const [result, setResult] = useState<RegisterResult | null>(null);
  const register = useRegister();
  const busy = register.isPending;

  // Both ways back keep the address in the field: "try again" is most often
  // after the owner approved it, and "a different address" after a typo.
  const startOver = () => {
    setResult(null);
    register.reset();
  };

  if (result?.outcome === "request-sent") {
    return <RequestSent email={result.email} onTryAgain={startOver} />;
  }
  if (result?.outcome === "check-email") {
    return <CheckEmail email={result.email} onDifferentAddress={startOver} />;
  }

  return (
    <AuthShell
      title="Create your account"
      footer="We’ll email you a link to confirm the address. You choose your password after that."
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!busy && email.trim()) register.mutate(email, { onSuccess: setResult });
        }}
        className="space-y-4"
      >
        <Field
          id="email"
          label="Email"
          type="email"
          hint="If you were invited, use the address your invitation was sent to. Otherwise we’ll ask for approval."
          value={email}
          onChange={(e) => {
            setEmail(e.target.value);
            if (register.isError) register.reset();
          }}
          disabled={busy}
          autoFocus
          autoComplete="username"
          invalid={register.isError}
          describedBy={register.isError ? "register-error" : undefined}
        />

        {register.isError && <FormError id="register-error" message={register.error.message} />}

        <SubmitButton busy={busy} busyLabel="Sending" disabled={!email.trim()}>
          Continue
        </SubmitButton>
      </form>

      <AuthLinks>
        <TextLink href="/">Already have an account? Sign in</TextLink>
      </AuthLinks>
    </AuthShell>
  );
}

function CheckEmail({ email, onDifferentAddress }: { email: string; onDifferentAddress: () => void }) {
  const resend = useResendConfirmation();
  // The server enforces the gap; the countdown only saves a pointless click.
  const [availableAt, setAvailableAt] = useState(() => Date.now() + RESEND_GAP_SECONDS * 1000);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const tick = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(tick);
  }, []);

  const wait = Math.max(0, Math.ceil((availableAt - now) / 1000));

  return (
    <AuthShell
      title="Check your email"
      footer="You can close this tab. The link in the email finishes the job from wherever you open it."
    >
      <Lede>
        We sent a link to <span className="font-medium break-all">{email}</span>. It works for 30
        minutes.
      </Lede>

      <AuthLinks>
        <TextButton onClick={onDifferentAddress}>Use a different address</TextButton>
        <TextButton
          disabled={wait > 0 || resend.isPending}
          onClick={() =>
            resend.mutate(email, {
              onSuccess: () => setAvailableAt(Date.now() + RESEND_GAP_SECONDS * 1000),
            })
          }
        >
          {resend.isPending ? "Sending" : wait > 0 ? `Send it again in ${wait}s` : "Send it again"}
        </TextButton>
      </AuthLinks>

      <div className="mt-3">
        {resend.isError && <FormError id="resend-error" message={resend.error.message} />}
        {resend.isSuccess && (
          <p role="status" className="text-sm text-muted-foreground">
            Sent again. Only the newest link will work.
          </p>
        )}
      </div>
    </AuthShell>
  );
}

function RequestSent({ email, onTryAgain }: { email: string; onTryAgain: () => void }) {
  return (
    <AuthShell title="Request sent">
      <Lede>
        Your request to use sparstrowgen with{" "}
        <span className="font-medium break-all">{email}</span> has been sent for approval.
      </Lede>
      <Lede>Once it’s approved, come back and create your account with the same address.</Lede>
      <AuthLinks>
        <TextLink href="/">Back to sign in</TextLink>
        <TextButton onClick={onTryAgain}>Try again</TextButton>
      </AuthLinks>
    </AuthShell>
  );
}
