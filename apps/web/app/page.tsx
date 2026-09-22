"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { ChatSurface } from "@/components/chat/chat-surface";
import { SignIn, SignInChecking, SignInUnreachable } from "@/components/auth/sign-in";
import { useSession, useSetup } from "@/lib/queries";
import { useSetupView } from "@/lib/store";

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
/* Someone who has never connected a computer meets setup rather than a Chat
   that cannot send (spec US2, D-046). Three things have to be true first, and
   all three are guards against sending the wrong person there:

     signed in      — otherwise there is no account to set up
     settled        — the machines list has actually loaded; an empty list while
                      the query is pending is indistinguishable from an account
                      with no computer (docs/Bugs.md B-46)
     not skipped    — read from this browser, and `loaded` says it was really
                      read rather than assumed

   It is an effect rather than a render-time redirect because Chat is the right
   thing to show while any of those is still unknown.

   The skip has one exception, and only one: an account with no workspace at
   all. Every conversation is kept in a workspace (D-050), so there is nothing
   for Chat to list and no folder for a new one to be started in — honouring a
   skip there would land somebody on a permanently empty screen instead of on
   the one box that fixes it. `setup.blocked` is that case and nothing else. */
function useFirstRunRedirect(signedIn: boolean) {
  const router = useRouter();
  const setup = useSetup();
  const { skipped, loaded, loadSkipped } = useSetupView();

  useEffect(() => loadSkipped(), [loadSkipped]);

  const wanted = !skipped || setup.blocked;
  const send = signedIn && loaded && wanted && setup.settled && !setup.complete;
  useEffect(() => {
    if (send) router.replace("/setup");
  }, [send, router]);
}

export default function Home() {
  const session = useSession();
  useFirstRunRedirect(session.data?.signedIn === true);

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
