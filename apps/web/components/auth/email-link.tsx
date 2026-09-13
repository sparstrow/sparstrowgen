"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import type { UseQueryResult } from "@tanstack/react-query";
import { buttonVariants } from "@/components/ui/button";
import type { EmailLink, EmailLinkKind } from "@/lib/api";
import {
  useCompletePasswordReset,
  useCompleteRegistration,
  useEmailLink,
} from "@/lib/queries";
import { SignInUnreachable } from "./sign-in";
import {
  AuthLinks,
  AuthShell,
  Field,
  FormError,
  Lede,
  ReadOnlyField,
  SubmitButton,
  TextLink,
} from "./shell";

/* The two pages an email link opens: finishing an account, and choosing a new
 * password.
 *
 * Both ask the server about the link BEFORE showing a form. Typing a password
 * into a link that expired an hour ago, and only then being told, is the
 * failure this avoids. A link that stops working while the form is open (a
 * newer email was sent from another tab) is caught on submit, and the page
 * falls back to the same explanation.
 */

const MIN_PASSWORD = 12;

export function ConfirmEmail({ token }: { token: string }) {
  const link = useEmailLink("verify", token);
  const complete = useCompleteRegistration();
  const router = useRouter();

  return (
    <EmailLinkGate kind="verify" link={link}>
      {(email) => (
        <PasswordForm
          title="Choose a password"
          label="Password"
          email={email}
          submitLabel="Create account"
          busyLabel="Creating your account"
          busy={complete.isPending}
          error={complete.isError ? complete.error.message : null}
          onEdit={() => complete.isError && complete.reset()}
          onSubmit={(password) =>
            complete.mutate(
              { token, password },
              {
                onSuccess: () => {
                  toast.success("Your account is ready.");
                  router.replace("/");
                },
              },
            )
          }
        />
      )}
    </EmailLinkGate>
  );
}

export function ResetPassword({ token }: { token: string }) {
  const link = useEmailLink("reset", token);
  const complete = useCompletePasswordReset();
  const router = useRouter();

  return (
    <EmailLinkGate kind="reset" link={link}>
      {(email) => (
        <PasswordForm
          title="Choose a new password"
          label="New password"
          email={email}
          submitLabel="Change password"
          busyLabel="Changing password"
          footer="Changing it signs out every other browser signed in to this account."
          busy={complete.isPending}
          error={complete.isError ? complete.error.message : null}
          onEdit={() => complete.isError && complete.reset()}
          onSubmit={(password) =>
            complete.mutate(
              { token, password },
              {
                onSuccess: () => {
                  toast.success("Password changed", {
                    description: "Every other browser signed in to this account has been signed out.",
                  });
                  router.replace("/");
                },
              },
            )
          }
        />
      )}
    </EmailLinkGate>
  );
}

function EmailLinkGate({
  kind,
  link,
  children,
}: {
  kind: EmailLinkKind;
  link: UseQueryResult<EmailLink>;
  children: (email: string) => React.ReactNode;
}) {
  if (link.isPending) {
    return (
      <main className="flex min-h-dvh items-center justify-center">
        <Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />
        <span className="sr-only">Checking this link</span>
      </main>
    );
  }
  if (link.isError) {
    return <SignInUnreachable error={link.error.message} onRetry={() => void link.refetch()} />;
  }
  if (!link.data.usable) return <LinkProblem kind={kind} link={link.data} />;
  return children(link.data.email);
}

function LinkProblem({
  kind,
  link,
}: {
  kind: EmailLinkKind;
  link: Extract<EmailLink, { usable: false }>;
}) {
  const alreadySetUp = kind === "verify" && link.reason === "used" && link.accountReady;

  const copy = alreadySetUp
    ? {
        title: "Your account is already set up",
        body: "This link has already been used, so there is nothing left to do with it.",
      }
    : link.reason === "superseded"
      ? {
          title: "A newer link was sent",
          body: "Only the most recent email works. Open the newest one, or ask for another.",
        }
      : link.reason === "used"
        ? {
            title: "This link has already been used",
            body:
              kind === "reset"
                ? "Each reset link works once. If you still need to change your password, ask for a new link."
                : "Each link works once.",
          }
        : { title: "This link has expired", body: "Links work once, for 30 minutes." };

  const action = alreadySetUp
    ? { href: "/", label: "Sign in" }
    : kind === "reset"
      ? { href: "/forgot", label: "Send a new reset link" }
      : { href: "/register", label: "Send a new link" };

  return (
    <AuthShell title={copy.title}>
      <Lede>{copy.body}</Lede>
      <Link href={action.href} className={buttonVariants({ className: "w-full" })}>
        {action.label}
      </Link>
      {!alreadySetUp && (
        <AuthLinks>
          <TextLink href="/">Back to sign in</TextLink>
        </AuthLinks>
      )}
    </AuthShell>
  );
}

function PasswordForm({
  title,
  label,
  email,
  submitLabel,
  busyLabel,
  footer,
  busy,
  error,
  onEdit,
  onSubmit,
}: {
  title: string;
  label: string;
  email: string;
  submitLabel: string;
  busyLabel: string;
  footer?: string;
  busy: boolean;
  error: string | null;
  onEdit: () => void;
  onSubmit: (password: string) => void;
}) {
  const [password, setPassword] = useState("");
  const tooShort = password.length > 0 && password.length < MIN_PASSWORD;

  return (
    <AuthShell title={title} footer={footer}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!busy && password.length >= MIN_PASSWORD) onSubmit(password);
        }}
        className="space-y-4"
      >
        <ReadOnlyField label="Email" value={email} />
        {/* For password managers: they save a new password against the
            username field nearest it, and a paragraph of text is not one. */}
        <input
          type="email"
          name="username"
          autoComplete="username"
          value={email}
          readOnly
          tabIndex={-1}
          aria-hidden
          className="sr-only"
        />

        <Field
          id="new-password"
          label={label}
          type="password"
          hint={`At least ${MIN_PASSWORD} characters. Length is what resists guessing, so a phrase you will remember beats a short one with symbols in it.`}
          value={password}
          onChange={(e) => {
            setPassword(e.target.value);
            onEdit();
          }}
          disabled={busy}
          autoFocus
          autoComplete="new-password"
          invalid={tooShort || error !== null}
          describedBy={error ? "password-error" : undefined}
        />

        {error && <FormError id="password-error" message={error} />}

        <SubmitButton busy={busy} busyLabel={busyLabel} disabled={password.length < MIN_PASSWORD}>
          {submitLabel}
        </SubmitButton>
      </form>
    </AuthShell>
  );
}
