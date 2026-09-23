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
import type { Conversation, Entry, Model, Provider, ProviderId, Workspace } from "./chat-types";
import { useWorkspaceView } from "./store";

/* Every read of server state goes through here, and every realtime event
   patches this cache rather than a second copy of the data (AGENTS.md §3). */

export const keys = {
  session: ["session"] as const,
  appearance: ["appearance"] as const,
  profile: ["profile"] as const,
  providers: ["providers"] as const,
  daemon: ["daemon"] as const,
  machines: ["machines"] as const,
  workspaces: ["workspaces"] as const,
  // The workspace is part of the key, not a filter applied afterwards: two
  // workspaces are two different lists, and caching them under one key would
  // show the last one's conversations for a moment after every switch.
  conversations: (workspaceId: string, q: string) =>
    ["conversations", workspaceId, q] as const,
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

/** The agents the workspace on screen can run.
 *
 *  Keyed by workspace, because it is the agents on the computer THIS
 *  workspace's work would go to (migration 00018). Offering one that the next
 *  message could not reach is worse than offering none.
 *
 *  No `initialData`. With it, an empty list counts as fresh data, and with an
 *  infinite staleTime that means this query never fetches at all — it only ever
 *  showed agents because the socket used to write them into it. Once the key
 *  gained the workspace, the socket's write landed on a different key and the
 *  agent menu opened empty for good (docs/Bugs.md B-48). Callers default the
 *  pending list to [] themselves. */
export function useProviders() {
  const workspaceId = useCurrentWorkspace();
  return useQuery<Provider[]>({
    queryKey: [...keys.providers, workspaceId] as const,
    queryFn: () => api.providers(workspaceId!),
    enabled: workspaceId !== null,
    // The socket says when the daemon's agents change (see "providers" below),
    // so polling would only duplicate a push we already get.
    staleTime: Infinity,
  });
}

/** Which of the account's computers a workspace may use, and which of a
 *  computer's workspaces offer it — the same row read from either end, because
 *  the owner asked to edit it from both. */
export function useWorkspaceMachines(workspaceId: string | null) {
  return useQuery({
    queryKey: ["workspace-machines", workspaceId] as const,
    queryFn: () => api.workspaceMachines(workspaceId!),
    enabled: workspaceId !== null,
  });
}

/** Whether the workspace on screen can run anything right now, and on what.
 *
 *  This used to be `useDaemon().online`, which answers for the whole ACCOUNT.
 *  Once a workspace could be given its own computers (00018) that became the
 *  wrong question: with the laptop online and the workspace given only the
 *  desktop, Send was offered and then refused (docs/Bugs.md B-49).
 *
 *  `reachable` follows the server's own rule (hub.Target) rather than
 *  re-deriving it: one of the workspace's computers is connected — or, in
 *  development only, the shared-token daemon is, which has no machine row and
 *  so shows up only as agents to run. */
export function useWorkspaceReach() {
  const workspaceId = useCurrentWorkspace();
  const machines = useWorkspaceMachines(workspaceId);
  const { data: providers } = useProviders();
  const assigned = (machines.data ?? []).filter((m) => m.assigned);
  const online = assigned.filter((m) => m.online);
  return {
    known: machines.isSuccess,
    assigned,
    online,
    reachable: online.length > 0 || (providers?.length ?? 0) > 0,
  };
}

export function useMachineWorkspaces(machineId: string | null) {
  return useQuery({
    queryKey: ["machine-workspaces", machineId] as const,
    queryFn: () => api.machineWorkspaces(machineId!),
    enabled: machineId !== null,
  });
}

/** Assigning, from either end.
 *
 *  Not optimistic. AGENTS.md §3's test asks whether the outcome is predictable,
 *  and this one is not the way a rename is: the server answers with the whole
 *  list, and what it decides changes where work can run. A tick that flips back
 *  is better than one that lies about which computer the next message reaches.
 *
 *  Both invalidate the providers list, because that is the visible consequence
 *  — take the last computer out of a workspace and its agents go with it. */
function useAssignmentInvalidation() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: ["workspace-machines"] });
    void qc.invalidateQueries({ queryKey: ["machine-workspaces"] });
    void qc.invalidateQueries({ queryKey: keys.providers });
  };
}

export function useSetWorkspaceMachine(workspaceId: string) {
  const qc = useQueryClient();
  const invalidate = useAssignmentInvalidation();
  return useMutation({
    mutationFn: ({ machineId, assigned }: { machineId: string; assigned: boolean }) =>
      api.setWorkspaceMachine(workspaceId, machineId, assigned),
    onSuccess: (list) => {
      qc.setQueryData(["workspace-machines", workspaceId], list);
      invalidate();
    },
    onError: (err: Error) =>
      toast.error("That computer could not be changed", { description: err.message }),
  });
}

export function useSetMachineWorkspace(machineId: string) {
  const qc = useQueryClient();
  const invalidate = useAssignmentInvalidation();
  return useMutation({
    mutationFn: ({ workspaceId, assigned }: { workspaceId: string; assigned: boolean }) =>
      api.setMachineWorkspace(machineId, workspaceId, assigned),
    onSuccess: (list) => {
      qc.setQueryData(["machine-workspaces", machineId], list);
      invalidate();
    },
    onError: (err: Error) =>
      toast.error("That workspace could not be changed", { description: err.message }),
  });
}

// ---------------------------------------------------------------------------
// workspaces
// ---------------------------------------------------------------------------

/** Every workspace this account can reach (docs/Decisions.md D-050).
 *
 *  An empty list is a real answer, not a failure: it is what a brand-new
 *  account has until first-run setup's second step makes the first one. */
export function useWorkspaces() {
  const session = useSession();
  return useQuery({
    queryKey: keys.workspaces,
    queryFn: api.workspaces,
    enabled: session.data?.signedIn ?? false,
    // Changes arrive over the websocket, so polling would duplicate a push.
    staleTime: Infinity,
  });
}

/** Which workspace this browser is looking at, resolved against what actually
 *  exists.
 *
 *  The remembered id is a hint and nothing more. A workspace can be renamed,
 *  and one day left or removed, and a browser holding an id it can no longer
 *  reach must land somewhere real rather than on a wall of 404s — so an id that
 *  is not in the list falls back to the first one. Null means the answer is not
 *  known yet, or this account has no workspace at all; either way nothing
 *  should be fetched for it. */
export function useCurrentWorkspace(): string | null {
  const workspaces = useWorkspaces();
  const remembered = useWorkspaceView((s) => s.workspaceId);
  const loaded = useWorkspaceView((s) => s.loaded);
  const load = useWorkspaceView((s) => s.loadWorkspace);

  // The server renders with nothing remembered, and only the browser can know
  // better — the same reason the rail's pin is read this way.
  useEffect(load, [load]);

  const list = workspaces.data;
  if (!loaded || !list || list.length === 0) return null;
  return list.some((w) => w.id === remembered) ? remembered : list[0].id;
}

/** The one this browser is in, as the whole workspace rather than its id. */
export function useWorkspace(): Workspace | null {
  const id = useCurrentWorkspace();
  const workspaces = useWorkspaces();
  return workspaces.data?.find((w) => w.id === id) ?? null;
}

export function useCreateWorkspace() {
  const qc = useQueryClient();
  const setWorkspace = useWorkspaceView((s) => s.setWorkspace);
  return useMutation({
    mutationFn: (name: string) => api.createWorkspace(name),
    onSuccess: (created) => {
      qc.setQueryData<Workspace[]>(keys.workspaces, (prev) =>
        [...(prev ?? []), created].sort((a, b) => a.name.localeCompare(b.name)),
      );
      // Making one is how you say you want to be in it. Creating a workspace
      // and staying in the old one would mean finding the new one in a menu
      // straight afterwards.
      setWorkspace(created.id);
    },
    onError: (err: Error) =>
      toast.error("Could not create the workspace", { description: err.message }),
  });
}

export function useRenameWorkspace() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => api.renameWorkspace(id, name),
    onSuccess: (saved) => {
      qc.setQueryData<Workspace[]>(keys.workspaces, (prev) =>
        (prev ?? [])
          .map((w) => (w.id === saved.id ? saved : w))
          .sort((a, b) => a.name.localeCompare(b.name)),
      );
    },
    onError: (err: Error) =>
      toast.error("Could not rename the workspace", { description: err.message }),
  });
}

export function useConversations(search: string) {
  const workspaceId = useCurrentWorkspace();
  return useQuery({
    queryKey: keys.conversations(workspaceId ?? "", search),
    queryFn: () => api.conversations(workspaceId!, search),
    // Nothing to list until we know which workspace. An account with none is
    // on its way to the setup wizard, not looking at an empty sidebar.
    enabled: workspaceId !== null,
    // Keeps the previous list on screen while a new search resolves, instead of
    // flashing empty on every keystroke. Only within one workspace: the key
    // includes the workspace, and carrying the old list across a SWITCH would
    // briefly show the other one's conversations, which is the one thing
    // workspaces exist to prevent.
    placeholderData: (prev, query) =>
      query?.queryKey[1] === workspaceId ? prev : undefined,
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

/** Invalidate conversation lists, whatever search term keyed them.
 *
 *  Scoped to one workspace when the caller knows which — a change in Work has
 *  nothing to say about the list in Personal. Given no workspace it refreshes
 *  all of them, which is right for a change we cannot place. */
function invalidateLists(qc: QueryClient, workspaceId?: string) {
  return qc.invalidateQueries({
    queryKey: workspaceId ? ["conversations", workspaceId] : ["conversations"],
  });
}

export function useCreateConversation() {
  const qc = useQueryClient();
  const workspaceId = useCurrentWorkspace();
  return useMutation({
    mutationFn: (input: { provider?: ProviderId; model?: Model; folder?: string }) => {
      // Never a default on the server's side (D-050). If this is reached with
      // no workspace the answer is to say so, not to put a transcript
      // somewhere nobody was looking.
      if (!workspaceId) throw new Error("No workspace is open yet.");
      return api.create({ ...input, workspaceId });
    },
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
  id: "profile" | "workspace" | "machines";
  /** The word in the stepper. */
  label: string;
  /** The sentence in the resume card. */
  task: string;
  done: boolean;
  /** A step setup cannot be left with outstanding. Only Workspace is: a name
   *  and a computer can wait, but every conversation has to be somewhere. */
  required?: boolean;
};

export type Setup = {
  steps: SetupStep[];
  done: number;
  total: number;
  /** Nothing outstanding. Not the same as "known": see `settled`. */
  complete: boolean;
  /** A required step is still outstanding, so setup cannot be skipped past.
   *  The redirect in app/page.tsx uses this to override a browser's saved
   *  skip: an account with no workspace has nothing to be shown instead. */
  blocked: boolean;
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
 *  The owner's order, in his words: "step 1 should be profile with adding
 *  avatar, setting name, bio, then step 2 should be workspace, and step 3 is
 *  mahcines."
 *
 *  Every step's doneness is DERIVED from what the account already has, so
 *  nothing tracks progress separately and nothing can disagree with reality: a
 *  profile is done when it has a name, and computers when there is one. A
 *  person who set their name on another device arrives with step one already
 *  complete, without anything having been synchronised. */
export function useSetup(): Setup {
  const machines = useMachines();
  const profile = useProfile();
  const workspaces = useWorkspaces();
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
      id: "workspace",
      label: "Workspace",
      task: "Make your first workspace",
      // Having one, not having two. A second workspace is what makes the
      // feature worth anything, but it is a thing to do when there is a second
      // body of work — not a hoop on the way in.
      done: (workspaces.data ?? []).length > 0,
      // The only step the app cannot run without: every conversation is in a
      // workspace, so there is nothing to skip to.
      required: true,
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
    blocked: steps.some((s) => s.required && !s.done),
    // Every query behind a step has to have answered. Until then an empty
    // profile, an empty workspace list and an empty machines list look exactly
    // like an account that has none of them (docs/Bugs.md B-46).
    settled: machines.isSuccess && profile.isSuccess && workspaces.isSuccess,
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
          // A signal, not a payload. The event carries the agents of the
          // ACCOUNT's current computer, while the list on screen is the
          // WORKSPACE's (00018) — writing one into the other would offer agents
          // the next message cannot reach. So every workspace's list is re-read
          // from the server, which knows which computer each would use.
          void qc.invalidateQueries({ queryKey: keys.providers });
          break;

        case "daemon":
          onDaemon(ev.online, ev.tooOld ?? false);
          break;

        case "machines":
          // Sent when a computer connects or drops, as well as when one is
          // assigned, so the workspace views of the same rows are re-read too —
          // they carry `online`, and the composer decides from them. And the
          // agents: taking a computer out of a workspace changes what it can
          // run without any "providers" event, because no daemon changed.
          void qc.invalidateQueries({ queryKey: ["machines"] });
          void qc.invalidateQueries({ queryKey: ["workspace-machines"] });
          void qc.invalidateQueries({ queryKey: ["machine-workspaces"] });
          void qc.invalidateQueries({ queryKey: keys.providers });
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

        case "workspaces":
          // Created or renamed, here or in another tab. The list is short and
          // the switcher shows it on every page, so it is re-read rather than
          // patched from two shapes of event.
          void qc.invalidateQueries({ queryKey: keys.workspaces });
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
          // Only the lists belonging to that conversation's OWN workspace.
          // Events are addressed to an account, so a Work conversation
          // finishing a turn arrives in a tab looking at Personal too, and
          // invalidating everything would make that tab refetch — harmlessly
          // today, and wrongly the moment anything is patched rather than
          // refetched.
          void invalidateLists(qc, ev.conversation.workspaceId);
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
