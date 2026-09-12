"use client";

import { ChatSurface } from "@/components/chat/chat-surface";
import { SignIn, SignInChecking, SignInUnreachable } from "@/components/auth/sign-in";
import { SignUp } from "@/components/auth/sign-up";
import { useSession } from "@/lib/queries";

/* Which of the three things this app is right now.
 *
 * Three and not two: a deployment that nobody has an account on is a different
 * situation from being signed out of one that exists, and showing a sign-in
 * form on a fresh install asks somebody to sign in to nothing. That is the
 * shape of it from the owner's side — he deploys, opens the URL, and the app
 * asks him to create the account rather than to guess a password that has never
 * been set.
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

  if (!session.data.claimed) return <SignUp />;
  if (!session.data.signedIn) return <SignIn />;

  return <ChatSurface />;
}
