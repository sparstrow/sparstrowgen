"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ChatSurface } from "@/components/chat/chat-surface";
import { SignIn, SignInChecking, SignInUnreachable } from "@/components/auth/sign-in";
import { keys, useSession } from "@/lib/queries";

/* Which of the two things this app is right now.
 *
 * The gate is here rather than in Next.js middleware on purpose. Middleware
 * protects PAGES, and the thing that must be protected is the Go API on its own
 * port — a browser that never loads this page can still call it. So the server
 * is the authority and this is only the surface: it asks who it is talking to
 * and renders accordingly, and every answer it gets is one the server enforced.
 */
export default function Home() {
  const qc = useQueryClient();
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

  if (!session.data) {
    return <SignIn onSignedIn={() => qc.setQueryData(keys.session, true)} />;
  }

  return <ChatSurface />;
}
