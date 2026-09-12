"use client";

import { useState } from "react";
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { api, NotSignedIn } from "@/lib/api";

/** One QueryClient per browser session, created in state rather than at module
 *  scope — a module-level client would be shared across requests during SSR and
 *  leak one user's data into another's render. */
export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(() => {
    /* A session can end while the app is open — it expires, or the owner signs
       out everywhere from his phone. When that happens EVERY request starts
       failing, and without this the screen fills with error toasts about
       conversations instead of saying the one true thing: you are signed out.

       Handled centrally rather than at each call site, because the call sites
       are every feature in the app and the next one added would forget. */
    const signedOut = (err: unknown) => {
      if (!(err instanceof NotSignedIn)) return;

      // ASK the server rather than asserting it here, and the reason is a race
      // that would otherwise be permanent. Signing in and changing a password
      // both replace the cookie, and a request that left under the OLD one can
      // land after that — a straggling 401 arriving a moment after a successful
      // sign-in. Writing "signed out" on the strength of it would throw the
      // owner back to the login screen while the browser holds a perfectly good
      // session, and `staleTime: Infinity` means nothing would ever correct it.
      //
      // Fetching instead lets the server settle it, and concurrent 401s collapse
      // into one request because the key is the same. staleTime 0 because the
      // cached answer is exactly what is in doubt.
      void client
        .fetchQuery({ queryKey: ["session"], queryFn: api.session, staleTime: 0 })
        .then((session) => {
          // Only once it is CONFIRMED: everything else was read with a session
          // that no longer exists, so none of it may sit behind the login form
          // or be handed to whoever signs in next.
          if (!session.signedIn) {
            client.removeQueries({ predicate: (q) => q.queryKey[0] !== "session" });
          }
        })
        .catch(() => {
          // The session endpoint is unreachable too. useSession surfaces that
          // as "can't reach the server", which is the honest answer and a
          // different screen from being signed out.
        });
    };
    const client: QueryClient = new QueryClient({
      queryCache: new QueryCache({ onError: signedOut }),
      mutationCache: new MutationCache({ onError: signedOut }),
      defaultOptions: {
        queries: {
          // Almost everything here is pushed over the websocket the moment it
          // changes, so background refetching would mostly re-fetch what we
          // already have. Refetch on reconnect is the exception: that is
          // precisely when the cache may have missed events.
          refetchOnWindowFocus: false,
          refetchOnReconnect: true,
          staleTime: 30_000,
          // Never retry "not signed in": it is an answer, not a blip, and
          // retrying only delays the login screen.
          retry: (count, err) => !(err instanceof NotSignedIn) && count < 1,
        },
      },
    });
    return client;
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
