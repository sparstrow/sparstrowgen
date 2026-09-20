"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { keys } from "@/lib/queries";

/* Connecting a computer, in one place.
 *
 * Two surfaces run this now — first-run setup and "Add computer" in Machines —
 * and the sequence is fiddly enough that two copies would drift: mint a
 * one-time request, open the `sparstrowgen://` link, poll until the daemon
 * spends it, then approve. Only the wrapping differs, so only the wrapping is
 * written twice (docs/Decisions.md D-046).
 *
 * There is no way to ask a computer whether the component is installed. The
 * daemon dials out only, so opening the link IS the check, and silence is
 * ambiguous: not installed, still starting, the browser blocked the handler, or
 * the claim was refused because another account holds that computer
 * (docs/KnownGaps.md G-40). After UNANSWERED_AFTER we say exactly that and
 * offer both ways forward — never that it is absent.
 */

/** Matches the copy: the daemon answers in about two seconds when it is there
 *  (docs/Capabilities.md), so eight is long enough to be fair to a slow start
 *  and short enough that nobody is left watching a spinner. */
const UNANSWERED_AFTER = 8_000;
const POLL_EVERY = 1_500;

export type PairStage = "idle" | "waiting" | "unanswered" | "approval" | "error";

export type Pairing = {
  stage: PairStage;
  /** The computer that claimed the request, once one has. Set on `approval`, so
   *  the prompt can name what is being approved rather than asking about "this
   *  computer" and hoping. */
  machineName: string | null;
  /** Set only on `error`; the reason the request could not be started. */
  error: string | null;
  /** True while approve is in flight, so the button can say "Connecting…". */
  approving: boolean;
  begin: () => void;
  /** Opens the same link again without minting a new request. */
  retry: () => void;
  approve: () => Promise<boolean>;
  notNow: () => void;
};

export function usePairing(): Pairing {
  const queryClient = useQueryClient();
  const [stage, setStage] = useState<PairStage>("idle");
  const [error, setError] = useState<string | null>(null);
  const [approving, setApproving] = useState(false);
  const [machineName, setMachineName] = useState<string | null>(null);
  // The request and its launch link are not rendered, and writing them to state
  // would restart the polling effect on every stage change.
  const active = useRef<{ id: string; uri: string } | null>(null);

  const reset = useCallback(() => {
    active.current = null;
    setMachineName(null);
    setStage("idle");
  }, []);

  const begin = useCallback(() => {
    setError(null);
    setStage("waiting");
    void (async () => {
      try {
        const next = await api.startPairing();
        active.current = { id: next.pairing.id, uri: next.launchUri };
        // Assigning the location hands the link to Windows, which opens the
        // installed daemon. It does not navigate this page away.
        window.location.assign(next.launchUri);
      } catch (e) {
        active.current = null;
        setError((e as Error).message);
        setStage("error");
      }
    })();
  }, []);

  const retry = useCallback(() => {
    const current = active.current;
    // A request lives ten minutes, so re-opening the link is right. Without one
    // — the page was reloaded mid-pairing — start over rather than do nothing.
    if (!current) return begin();
    setStage("waiting");
    window.location.assign(current.uri);
  }, [begin]);

  const approve = useCallback(async () => {
    const current = active.current;
    if (!current) return false;
    setApproving(true);
    try {
      await api.approvePairing(current.id);
      await queryClient.invalidateQueries({ queryKey: keys.machines });
      reset();
      return true;
    } catch (e) {
      setError((e as Error).message);
      setStage("error");
      return false;
    } finally {
      setApproving(false);
    }
  }, [queryClient, reset]);

  const notNow = useCallback(() => {
    const current = active.current;
    reset();
    if (!current) return;
    // Best effort: the request expires on its own in ten minutes either way, so
    // a failure here costs nothing the person can see.
    void api
      .declinePairing(current.id)
      .then(() => queryClient.invalidateQueries({ queryKey: keys.machines }))
      .catch(() => {});
  }, [queryClient, reset]);

  // Polling continues through `unanswered`, and this is the whole point of that
  // state: it says the computer "may still be starting", so we have to still be
  // listening when it does. Only the countdown belongs to `waiting` — giving up
  // at eight seconds would make the sentence a lie and leave a daemon that
  // claimed the request at nine seconds waiting for an approval nobody is
  // being offered.
  const listening = stage === "waiting" || stage === "unanswered";

  // The countdown is its own effect, keyed on `waiting`, so that pressing "Try
  // again" starts it again. Folded into the poll below — which must NOT be torn
  // down when waiting becomes unanswered — it would only ever run once.
  useEffect(() => {
    if (stage !== "waiting") return;
    const timeout = window.setTimeout(
      () => setStage((s) => (s === "waiting" ? "unanswered" : s)),
      UNANSWERED_AFTER,
    );
    return () => window.clearTimeout(timeout);
  }, [stage]);

  useEffect(() => {
    if (!listening) return;
    const poll = window.setInterval(() => {
      const current = active.current;
      if (!current) return;
      void (async () => {
        try {
          const p = await api.pairing(current.id);
          if (p.status === "claimed") {
            setMachineName(p.machineName ?? null);
            setStage("approval");
          }
          else if (p.status === "approved") {
            // Already this account's computer: the daemon recognised its own
            // credential, so there is nothing to approve.
            await queryClient.invalidateQueries({ queryKey: keys.machines });
            reset();
          } else if (p.status === "rejected") reset();
        } catch (e) {
          setError((e as Error).message);
          setStage("error");
        }
      })();
    }, POLL_EVERY);
    return () => window.clearInterval(poll);
    // `listening` rather than `stage`, so moving from waiting to unanswered
    // does not tear the poll down and start it again.
  }, [listening, queryClient, reset]);

  return { stage, machineName, error, approving, begin, retry, approve, notNow };
}
