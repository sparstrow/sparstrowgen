"use client";

import { SetupWizard } from "@/components/setup/setup-wizard";
import { SignIn, SignInChecking, SignInUnreachable } from "@/components/auth/sign-in";
import { useSession } from "@/lib/queries";

/* First-run setup has a route of its own so it can be returned to, linked to,
   and left. The same gate as the other signed-in pages: the server is the
   authority and this only renders what it is allowed to see (D-025). */
export default function SetupPage() {
  const session = useSession();

  if (session.isPending) return <SignInChecking />;
  if (session.isError) {
    return (
      <SignInUnreachable
        error={session.error.message}
        onRetry={() => void session.refetch()}
      />
    );
  }
  if (!session.data.signedIn) return <SignIn />;

  return <SetupWizard />;
}
