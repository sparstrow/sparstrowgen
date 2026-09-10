"use client";

import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

/** One QueryClient per browser session, created in state rather than at module
 *  scope — a module-level client would be shared across requests during SSR and
 *  leak one user's data into another's render. */
export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // Almost everything here is pushed over the websocket the moment it
            // changes, so background refetching would mostly re-fetch what we
            // already have. Refetch on reconnect is the exception: that is
            // precisely when the cache may have missed events.
            refetchOnWindowFocus: false,
            refetchOnReconnect: true,
            staleTime: 30_000,
            retry: 1,
          },
        },
      }),
  );
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
