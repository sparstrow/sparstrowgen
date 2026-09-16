"use client";

import { useEffect } from "react";
import { ThemeProvider, useTheme } from "next-themes";
import { useSession } from "@/lib/queries";
import {
  DEFAULT_APPEARANCE,
  applyAppearance,
  cacheAppearance,
  cachedAppearance,
  readable,
} from "@/lib/appearance";

/* The account decides how the app looks; this keeps the document in step with
   it (spec 2026-09-12-appearance-preferences).

   Two sources, and the order matters. Before the session is known, the page
   wears whatever this browser saw last — written by the first-paint script, so
   there is no wrong-theme flash. Once the session answers, the account's saved
   appearance wins and is remembered for next time.

   Signed out, nothing is overwritten: the sign-in screen keeps the last look
   this browser had, or the product default on a browser that has none. */
function FollowsTheAccount({ children }: { children: React.ReactNode }) {
  const session = useSession();
  const { setTheme } = useTheme();
  const saved = session.data?.signedIn ? session.data.appearance : undefined;

  useEffect(() => {
    const appearance = saved ? readable(saved) : cachedAppearance();
    applyAppearance(appearance);
    setTheme(appearance.mode);
    if (saved) cacheAppearance(appearance);
  }, [saved, setTheme]);

  return <>{children}</>;
}

export function AppearanceProvider({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme={DEFAULT_APPEARANCE.mode}
      enableSystem
      // Colours are tokens, so a mode change is a repaint rather than a
      // re-layout; the transition suppression next-themes does by default
      // makes the switch look abrupt rather than smooth.
      disableTransitionOnChange={false}
    >
      <FollowsTheAccount>{children}</FollowsTheAccount>
    </ThemeProvider>
  );
}
