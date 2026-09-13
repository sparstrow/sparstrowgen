"use client";

import { ChatSurface } from "@/components/chat/chat-surface";
import { SignIn, SignInChecking, SignInUnreachable } from "@/components/auth/sign-in";
import { useSession } from "@/lib/queries";

/* Which of the things this app is right now: checking, unreachable, signed out,
 * or signed in.
 *
 * Creating an account is not one of them. It has its own page (/register),
 * reached from sign in, because it begins with an email rather than with this
 * browser — the link that finishes it can be opened anywhere.
 *
 * The gate is here rather than in Next.js middleware on purpose. Middleware
 * protects PAGES, and the thing that must be protected is the Go API on its own
 * port — a browser that never loads this page can still call it. So the server
 * is the authority and this is only the surface: it asks what it is allowed to
 * see and renders accordingly, and every answer it gets is one the server
 * enforced (docs/Decisions.md D-025).
 */
export default function Home() {
  const session = useSession();

  if (session.isPending) return <SignInChecking />;

  // The server could not be reached at all, which is not the same as being
  // signed out and must not be shown as one — no password would fix it.
  if (session.isError) {
    return (
      <SignInUnreachable
        error={session.error.message}
        onRetry={() => void session.refetch()}
      />
    );
  }

  if (!session.data.signedIn) return <SignIn />;

  return <ChatSurface />;
}
