"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
import { useEffect } from "react";
import { toast } from "sonner";
import {
  accountAccess,
  api,
  connect,
  type EmailLinkKind,
  type ServerEvent,
  type Session,
} from "./api";
import type { Appearance } from "./api";
import type { Conversation, Entry, Model, Provider, ProviderId } from "./chat-types";

/* Every read of server state goes through here, and every realtime event
   patches this cache rather than a second copy of the data (AGENTS.md §3). */

export const keys = {
  session: ["session"] as const,
  appearance: ["appearance"] as const,
  profile: ["profile"] as const,
  providers: ["providers"] as const,
  daemon: ["daemon"] as const,
  machines: ["machines"] as const,
  conversations: (q: string) => ["conversations", q] as const,
  conversation: (id: string) => ["conversation", id] as const,
  emailLink: (kind: EmailLinkKind, token: string) => ["email-link", kind, token] as const,
};

/** Whether this browser is signed in, and as whom.
 *
 *  Asked once before anything else, so the app shows the right door rather than
 *  a chat surface whose every request fails. Not retried: "not signed in" is an
 *  answer, and retrying it three times only delays the screen that lets the
 *  owner do something about it. */
export function useSession() {
  return useQuery({
    queryKey: keys.session,
    queryFn: api.session,
    staleTime: Infinity,
    retry: false,
    // Asked again whenever this tab is looked at. The live connection announces
    // an appearance change to the pages that hold one (Chat, Settings →
    // Updates); every other page has none, and without this a tab left open on
    // Machines would keep the look it was loaded with. "always" rather than
    // true, because staleTime above means it is never stale.
    refetchOnWindowFocus: "always",
  });
}

/** How this account wants the app to look.
 *
 *  The session already carried it, so the settings screen opens on the real
 *  choice rather than a skeleton; this query is what re-reads it afterwards and
 *  what another tab converges on. */
export function useAppearance() {
  const session = useSession();
  const signedIn = session.data?.signedIn ?? false;
  return useQuery({
    queryKey: keys.appearance,
    queryFn: api.appearance,
    initialData: signedIn ? session.data?.appearance : undefined,
    enabled: signedIn,
    // Same reason as the session above: a settings screen left open in another
    // tab shows the account's real choice as soon as it is looked at.
    refetchOnWindowFocus: "always",
  });
}

/** Saves all three choices together.
 *
 *  Optimistic, which is the right call here by AGENTS.md §3's test: the outcome
 *  is predictable, nothing navigates, and undoing it is putting back the values
 *  we already hold. The session copy is updated too, because that is what the
 *  provider paints from — so the change is seen at once rather than after a
 *  round trip. A failure puts both back and says so. */
export function useSaveAppearance() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (next: Appearance) => api.saveAppearance(next),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: keys.appearance });
      const previous = qc.getQueryData<Appearance>(keys.appearance);
      const previousSession = qc.getQueryData<Session>(keys.session);
      qc.setQueryData(keys.appearance, next);
      if (previousSession?.signedIn) {
        qc.setQueryData<Session>(keys.session, { ...previousSession, appearance: next });
      }
      return { previous, previousSession };
    },
    onError: (error, _next, context) => {
      if (context?.previous) qc.setQueryData(keys.appearance, context.previous);
      if (context?.previousSession) qc.setQueryData(keys.session, context.previousSession);
      toast.error("Appearance was not saved", { description: (error as Error).message });
    },
    onSuccess: (saved) => {
      qc.setQueryData(keys.appearance, saved);
      const session = qc.getQueryData<Session>(keys.session);
      if (session?.signedIn) qc.setQueryData<Session>(keys.session, { ...session, appearance: saved });
    },
  });
}

/** Writes a signed-in session into the cache without a round trip.
 *
 *  Signing up, signing in and changing a password all end with the server
 *  having just told us who this is, so asking it again immediately is a request
 *  whose answer we already hold. */
function useSessionWriter() {
  const qc = useQueryClient();
  return (email: string) =>
    qc.setQueryData<Session>(keys.session, { signedIn: true, email });
}

export function useSignIn() {
  const signedIn = useSessionWriter();
  return useMutation({
    mutationFn: (input: { email: string; password: string }) =>
      api.signIn(input.email, input.password),
    onSuccess: (email) => signedIn(email),
  });
}

/** Changing the password, which on the server ends every session and issues
 *  this browser a new one.
 *
 *  So there is nothing to invalidate and nowhere to send the owner: he stays on
 *  the screen he was on, holding a cookie he did not have a moment ago, and
 *  every other device is now signed out. */
export function useChangePassword() {
  return useMutation({
    mutationFn: (input: { current: string; next: string }) =>
      api.changePassword(input.current, input.next),
  });
}

export function useRegister() {
  return useMutation({ mutationFn: (email: string) => accountAccess.register(email) });
}

export function useResendConfirmation() {
  return useMutation({ mutationFn: (email: string) => accountAccess.resendConfirmation(email) });
}

/** Whether an emailed link still works. Asked once per visit and never retried:
 *  "expired" is an answer, not a blip. */
export function useEmailLink(kind: EmailLinkKind, token: string) {
  return useQuery({
    queryKey: keys.emailLink(kind, token),
    queryFn: () => accountAccess.emailLink(kind, token),
    staleTime: Infinity,
    retry: false,
  });
}

/** Finishing an account from its confirmation link.
 *
 *  A failure re-asks the server about the link, so a link that died while the
 *  form was open (a newer email sent from another tab) turns the page into the
 *  explanation instead of leaving a form that can never succeed. */
export function useCompleteRegistration() {
  const qc = useQueryClient();
  const signedIn = useSessionWriter();
  return useMutation({
    mutationFn: (input: { token: string; password: string }) =>
      accountAccess.completeRegistration(input.token, input.password),
    // The response carried the new session's cookie, so the answer to "who is
    // this browser" is already known.
    onSuccess: (email) => signedIn(email),
    onError: (_err, { token }) =>
      qc.invalidateQueries({ queryKey: keys.emailLink("verify", token) }),
  });
}

export function useRequestPasswordReset() {
  return useMutation({ mutationFn: (email: string) => accountAccess.requestPasswordReset(email) });
}

export function useCompletePasswordReset() {
  const qc = useQueryClient();
  const signedIn = useSessionWriter();
  return useMutation({
    mutationFn: (input: { token: string; password: string }) =>
      accountAccess.completePasswordReset(input.token, input.password),
    onSuccess: (email) => signedIn(email),
    onError: (_err, { token }) =>
      qc.invalidateQueries({ queryKey: keys.emailLink("reset", token) }),
  });
}

export function useSignOut() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (everywhere: boolean) => api.signOut(everywhere),
    // onError, not onSettled, for the cache changes below. Signing out is the
    // one action where pretending it worked is worse than reporting that it
    // did not: "sign out everywhere" is pressed BECAUSE something may be in
    // the wrong hands, and showing the login screen over sessions that are
    // still live would end the owner's worry without ending the exposure.
    onError: (err: Error, everywhere) =>
      toast.error(
        everywhere ? "Could not sign out everywhere" : "Could not sign out",
        {
          description: everywhere
            ? `${err.message}. Your other devices may still be signed in.`
            : err.message,
        },
      ),
    onSuccess: () => {
      // ORDER MATTERS, and getting it wrong looks like sign-out being broken
      // (docs/Bugs.md B-14).
      //
      // The session flag goes first. That re-renders the gate, which swaps the
      // chat surface for the login form and unmounts every component still
      // watching a query. Only then is it safe to drop the rest of the cache.
      //
      // Doing it the other way round — clear() and then set the flag — removes
      // the query objects that the still-mounted observers are subscribed to,
      // so they are left watching nothing and no re-render is ever triggered.
      // The request succeeds, the cookie is gone, and the screen sits there
      // showing a conversation the server would now refuse to hand over.
      //
      qc.setQueryData<Session>(keys.session, { signedIn: false });
      // Everything else was read with a session that no longer exists, so none
      // of it may stay behind the login form.
      qc.removeQueries({
        predicate: (q) => q.queryKey[0] !== keys.session[0],
      });
    },
  });
}

export function useProviders() {
  return useQuery({
    queryKey: keys.providers,
    queryFn: api.providers,
    // Providers arrive over the websocket the moment the daemon reports them,
    // so polling would only duplicate a push we already get.
    staleTime: Infinity,
    initialData: [] as Provider[],
  });
}

export function useConversations(search: string) {
  return useQuery({
    queryKey: keys.conversations(search),
    queryFn: () => api.conversations(search),
    // Keeps the previous list on screen while a new search resolves, instead of
    // flashing empty on every keystroke.
    placeholderData: (prev) => prev,
  });
}

export function useConversation(id: string | null) {
  return useQuery({
    queryKey: keys.conversation(id ?? ""),
    queryFn: () => api.conversation(id!),
    enabled: id !== null,
  });
}

// ---------------------------------------------------------------------------
// mutations
// ---------------------------------------------------------------------------

/** Invalidate every conversation list, whatever search term keyed it. */
function invalidateLists(qc: QueryClient) {
  return qc.invalidateQueries({ queryKey: ["conversations"] });
}

export function useCreateConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { provider?: ProviderId; model?: Model; folder?: string }) =>
      api.create(input),
    onSuccess: (created) => {
      qc.setQueryData(keys.conversation(created.id), created);
      void invalidateLists(qc);
    },
    onError: (err: Error) =>
      toast.error("Could not start a conversation", { description: err.message }),
  });
}

/* Rename and archive are optimistic: the outcome is predictable, the user stays
   where they are, failure is rare, and rollback is one cache write — which is
   the whole test AGENTS.md §3 sets for doing it. Sending a message is not
   optimistic, because what comes back is a model's answer and nothing about it
   is predictable. */
function useOptimisticPatch<
  T extends { title?: string; archived?: boolean; folder?: string },
>(
  apply: (c: Conversation, patch: T) => Conversation,
  failure: string,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: T }) => api.patch(id, patch),
    onMutate: async ({ id, patch }) => {
      await qc.cancelQueries({ queryKey: ["conversations"] });
      const lists = qc.getQueriesData<Conversation[]>({ queryKey: ["conversations"] });
      const detail = qc.getQueryData<Conversation>(keys.conversation(id));

      for (const [key, list] of lists) {
        if (!list) continue;
        qc.setQueryData(
          key,
          list.map((c) => (c.id === id ? apply(c, patch) : c)),
        );
      }
      if (detail) qc.setQueryData(keys.conversation(id), apply(detail, patch));

      return { lists, detail, id };
    },
    onError: (err: Error, _vars, ctx) => {
      // Put back exactly what was there, rather than refetching and hoping.
      for (const [key, list] of ctx?.lists ?? []) qc.setQueryData(key, list);
      if (ctx?.detail) qc.setQueryData(keys.conversation(ctx.id), ctx.detail);
      toast.error(failure, { description: err.message });
    },
    onSettled: () => invalidateLists(qc),
  });
}

export function useRenameConversation() {
  return useOptimisticPatch<{ title: string }>(
    (c, p) => ({ ...c, title: p.title }),
    "Rename failed",
  );
}

export function useArchiveConversation() {
  return useOptimisticPatch<{ archived: boolean }>(
    (c, p) => ({ ...c, archived: p.archived }),
    "Could not archive",
  );
}

/** Moving a conversation to another working directory.
 *
 *  Optimistic like the others: the outcome is predictable, the owner stays put,
 *  and a failure rolls back to a folder that is still on screen. */
export function useSetFolder() {
  return useOptimisticPatch<{ folder: string }>(
    (c, p) => ({ ...c, folder: p.folder }),
    "Could not change the folder",
  );
}

export function useDeleteConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.remove(id),
    onSuccess: (_v, id) => {
      qc.removeQueries({ queryKey: keys.conversation(id) });
      void invalidateLists(qc);
    },
    onError: (err: Error) =>
      toast.error("Could not delete", { description: err.message }),
  });
}

export function useSendMessage() {
  return useMutation({
    mutationFn: ({
      id,
      text,
      provider,
      model,
    }: {
      id: string;
      text: string;
      provider: ProviderId;
      model: Model;
    }) => api.send(id, { text, provider, model }),
  });
}

/** Ends a turn that is already running.
 *
 *  Nothing is patched here and no cache is invalidated: the turn ends when the
 *  daemon says it has, and that arrives as the same entry_done every other
 *  ending arrives as. Finishing it optimistically would mean guessing what text
 *  survived, and the whole reason to stop a turn is that you want to keep what
 *  it already said. */
export function useStopTurn() {
  return useMutation({ mutationFn: (turnId: string) => api.stopTurn(turnId) });
}

// ---------------------------------------------------------------------------
// realtime
// ---------------------------------------------------------------------------

/** Patches the Query cache from websocket events.
 *
 *  Entries are patched in place rather than triggering a refetch: a delta
 *  arrives every few characters, and refetching a whole transcript on each one
 *  would be absurd. Conversation-level changes invalidate, because they are
 *  rare and the server's version is authoritative. */
/** What the app knows about its two connections.
 *
 *  `serverConnected` is this browser's own socket: when it is down we are not
 *  being told anything, so nothing else here can be trusted to be current.
 *  `online` is the owner's computer, which only the server can report. They are
 *  separate facts and the header says different words for each — claiming a
 *  computer is online while we cannot hear the server would be a lie the app
 *  has no way to notice. */
export type DaemonState = { online: boolean; tooOld: boolean; serverConnected: boolean };

const DAEMON_INITIAL: DaemonState = { online: false, tooOld: false, serverConnected: false };

/** Read by every surface that shows the live status. This is server state, so
 *  it lives in the Query cache and the socket patches it — never mirrored into
 *  Zustand (AGENTS.md §3). */
export function useDaemon(): DaemonState {
  const { data } = useQuery({
    queryKey: keys.daemon,
    // Nothing fetches it; the socket is its only writer. queryFn exists so the
    // cache entry has a shape before the first event arrives.
    queryFn: () => DAEMON_INITIAL,
    staleTime: Infinity,
    gcTime: Infinity,
  });
  return data ?? DAEMON_INITIAL;
}

/** The paired computers. Loaded on every page now, not only on Machines: the
 *  header's live status names the computer, so it needs the list wherever it is
 *  shown. Realtime `machines` events invalidate it. */
export function useMachines() {
  return useQuery({ queryKey: keys.machines, queryFn: api.machines });
}

/** This account's name, description and whether it has a picture. Read
 *  wherever a person is shown, so it is one query the socket invalidates. */
export function useProfile() {
  return useQuery({ queryKey: keys.profile, queryFn: api.profile });
}

export function useSaveProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.saveProfile,
    // The server answers with the saved profile, so the cache takes it rather
    // than re-fetching what we were just handed.
    onSuccess: (saved) => qc.setQueryData(keys.profile, saved),
  });
}

export function useSaveAvatar() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.saveAvatar,
    onSuccess: (saved) => qc.setQueryData(keys.profile, saved),
  });
}

export function useRemoveAvatar() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.removeAvatar,
    onSuccess: (saved) => qc.setQueryData(keys.profile, saved),
  });
}

export type SetupStep = {
  id: "profile" | "machines";
  /** The word in the stepper. */
  label: string;
  /** The sentence in the resume card. */
  task: string;
  done: boolean;
};

export type Setup = {
  steps: SetupStep[];
  done: number;
  total: number;
  /** Nothing outstanding. Not the same as "known": see `settled`. */
  complete: boolean;
  /** The answer is based on a loaded machines list rather than a guess. Until
   *  this is true, nothing should route anyone anywhere or claim a step is
   *  undone — an empty list while the query is pending looks identical to an
   *  account with no computer (docs/Bugs.md B-46). */
  settled: boolean;
};

/** The one list of setup steps. The wizard renders it full-screen and the card
 *  in the Chat pane renders it compact; neither decides for itself what is
 *  outstanding (docs/Decisions.md D-046, D-047).
 *
 *  The owner's order is Profile, then Workspace, then Machines. Workspace is
 *  not built — it needs its own spec — so the list is Profile and Machines, and
 *  Workspace slots in here when it exists without either surface being touched.
 *
 *  Every step's doneness is DERIVED from what the account already has, so
 *  nothing tracks progress separately and nothing can disagree with reality: a
 *  profile is done when it has a name, and computers when there is one. A
 *  person who set their name on another device arrives with step one already
 *  complete, without anything having been synchronised. */
export function useSetup(): Setup {
  const machines = useMachines();
  const profile = useProfile();
  const steps: SetupStep[] = [
    {
      id: "profile",
      label: "Profile",
      task: "Add your name and picture",
      // A name is the one part that makes a difference to anyone else, so it is
      // what "done" means. A picture and a description are optional and always
      // will be — requiring them would make skipping the only way past.
      done: (profile.data?.displayName ?? "") !== "",
    },
    {
      id: "machines",
      label: "Machines",
      task: "Connect this computer",
      done: (machines.data ?? []).length > 0,
    },
  ];
  const done = steps.filter((s) => s.done).length;
  return {
    steps,
    done,
    total: steps.length,
    complete: done === steps.length,
    // Every query behind a step has to have answered. Until then an empty
    // profile and an empty machines list look exactly like an account that has
    // neither (docs/Bugs.md B-46).
    settled: machines.isSuccess && profile.isSuccess,
  };
}

/** Opens the one websocket. Mounted exactly once, by the app shell: `connect`
 *  makes a socket per call, so a second caller would mean a second connection. */
export function useRealtime() {
  const qc = useQueryClient();

  useEffect(() => {
    const setDaemon = (patch: Partial<DaemonState>) =>
      qc.setQueryData<DaemonState>(keys.daemon, (prev) => ({
        ...(prev ?? DAEMON_INITIAL),
        ...patch,
      }));
    const onDaemon = (online: boolean, tooOld = false) =>
      setDaemon({ online, tooOld: online && tooOld });
    const patchEntries = (
      conversationId: string,
      fn: (entries: Entry[]) => Entry[],
    ) =>
      qc.setQueryData<Conversation>(keys.conversation(conversationId), (prev) =>
        prev ? { ...prev, entries: fn(prev.entries) } : prev,
      );

    const handle = (ev: ServerEvent) => {
      switch (ev.type) {
        case "providers":
          qc.setQueryData(keys.providers, ev.providers);
          break;

        case "daemon":
          onDaemon(ev.online, ev.tooOld ?? false);
          break;

        case "machines":
          void qc.invalidateQueries({ queryKey: ["machines"] });
          break;

        case "appearance":
          // Both, because the provider paints from the session's copy and the
          // settings screen reads the other. Re-read rather than carry the
          // values on the event: the account is the one source of truth.
          void qc.invalidateQueries({ queryKey: keys.appearance });
          void qc.invalidateQueries({ queryKey: keys.session });
          break;

        case "profile":
          // The name and picture are in the shell on every page, so a tab that
          // was open while they changed would otherwise keep the old ones.
          void qc.invalidateQueries({ queryKey: keys.profile });
          break;

        case "conversation":
          qc.setQueryData<Conversation>(
            keys.conversation(ev.conversation.id),
            (prev) =>
              // The list payload carries no entries. Keeping the open
              // transcript is what stops a usage update blanking the screen.
              prev
                ? { ...ev.conversation, entries: prev.entries, seenBy: prev.seenBy }
                : prev,
          );
          void invalidateLists(qc);
          break;

        case "entry_added":
          patchEntries(ev.conversationId, (entries) =>
            entries.some((e) => e.id === ev.entry.id)
              ? entries
              : [...entries, ev.entry],
          );
          break;

        case "entry_delta":
          patchEntries(ev.conversationId, (entries) =>
            entries.map((e) =>
              e.id === ev.entryId && e.role === "agent"
                ? ({ ...e, text: (e.text ?? "") + ev.text } as Entry)
                : e,
            ),
          );
          break;

        case "entry_done":
          patchEntries(ev.conversationId, (entries) =>
            entries.map((e) => (e.id === ev.entry.id ? ev.entry : e)),
          );
          qc.setQueryData<Conversation>(
            keys.conversation(ev.conversationId),
            (prev) =>
              prev
                ? {
                    ...prev,
                    // The provider has now seen everything up to its own reply,
                    // so a switch away and back replays only what follows.
                    seenBy: { ...prev.seenBy, [ev.entry.provider]: prev.entries.length },
                  }
                : prev,
          );
          if (ev.entry.failure) {
            toast.error("Turn did not finish", { description: ev.entry.failure });
          }
          void invalidateLists(qc);
          break;
      }
    };

    return connect(handle, (open) => {
      // The socket being up says the server is reachable. Whether the owner's
      // machine is reachable is a separate fact the server tells us — and while
      // the socket is down we are told nothing, so the computer stops counting
      // as reachable until the server says otherwise again.
      setDaemon(open ? { serverConnected: true } : { serverConnected: false, online: false, tooOld: false });
    });
  }, [qc]);
}
