"use client";

import { useState } from "react";
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { NotSignedIn } from "@/lib/api";

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
      if (err instanceof NotSignedIn) client.setQueryData(["session"], false);
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
