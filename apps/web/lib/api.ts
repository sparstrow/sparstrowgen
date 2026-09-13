import type {
  AgentMessage,
  Conversation,
  DirListing,
  Entry,
  Model,
  Provider,
  ProviderId,
} from "./chat-types";

/* The server owns every shape here. Its Go structs in
   server/internal/protocol/protocol.go carry json tags matching chat-types.ts,
   and until Protobuf lands (docs/Decisions.md D-017) the two are hand-kept in
   step — change one and change the other in the same commit. */

const BASE =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

/** Every call to the API carries the session cookie.
 *
 *  Wrapped once rather than passed at each of a dozen call sites, because
 *  forgetting it on ONE of them is a silent 401 on one feature — the kind of
 *  bug a person finds, months later, and cannot reproduce.
 *
 *  `include` and not `same-origin`: in development the app is on :3000 and the
 *  API on :8080, which is a different ORIGIN even though it is the same site.
 *  In production a proxy puts them on one origin and this costs nothing. */
function request(url: string, init: RequestInit = {}) {
  return fetch(url, { ...init, credentials: "include" });
}

/** POSTs JSON and throws the server's own wording on failure.
 *
 *  The thing that actually changed — the session cookie — is set by the browser
 *  out of reach of this code, so what comes back is only what the server wants
 *  to say about the account. */
async function post<T>(url: string, body: unknown): Promise<T> {
  const res = await request(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const failed = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new Error(failed?.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

/** What every path that starts a session answers with.
 *
 *  The email comes back from the SERVER rather than being echoed from the form,
 *  because the server normalises it — trimmed and lowercased — and the account
 *  menu should show the address that exists, not the one that was typed. */
type SignedIn = { ok: true; email: string };

/** Thrown when the server says "not signed in", so the app can show the login
 *  screen rather than an error toast about a conversation list. */
export class NotSignedIn extends Error {
  constructor() {
    super("not signed in");
    this.name = "NotSignedIn";
  }
}

async function json<T>(res: Response): Promise<T> {
  if (res.status === 401) throw new NotSignedIn();
  if (!res.ok) {
    // The server sends {"error": "..."} and its wording is written to be read
    // by the owner, so it is surfaced rather than replaced with a generic one.
    const body = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new Error(body?.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

/** Who the server thinks this browser is.
 *
 *  `email` is present only when signed in, which is why it is optional rather
 *  than an empty string — an empty string would render as a blank account menu
 *  instead of an obviously missing one. */
export type Session = {
  signedIn: boolean;
  email?: string;
};

/* Account access (US1). Served by server/internal/api/accounts.go; the data
   contract is design-system/designs/Accounts/account-access.handoff.md. */

/** What registering an address led to. The server decides; the screen reports.
 *  An address that already has an account answers "check-email" too — the email
 *  says it exists — so the response never reveals which addresses are
 *  registered. `email` is the address as the server normalised it. */
export type RegisterResult = {
  outcome: "check-email" | "request-sent";
  email: string;
};

export type EmailLinkKind = "verify" | "reset";

/** What a link from an email is still good for, asked before showing a form.
 *  An unknown token is reported as expired rather than as its own case. */
export type EmailLink =
  | { usable: true; email: string }
  | {
      usable: false;
      reason: "expired" | "used" | "superseded";
      /** For a used confirmation link: the account it created exists. */
      accountReady: boolean;
    };

export type AccountAccess = {
  register(email: string): Promise<RegisterResult>;
  resendConfirmation(email: string): Promise<void>;
  emailLink(kind: EmailLinkKind, token: string): Promise<EmailLink>;
  /** Resolves with the new account's email. The response also sets the session
   *  cookie, so this browser is signed in when it resolves. */
  completeRegistration(token: string, password: string): Promise<string>;
  /** Resolves with the normalised address whether or not it has an account. */
  requestPasswordReset(email: string): Promise<string>;
  /** Resolves with the account's email. Every other session is ended and this
   *  browser is given a new one. */
  completePasswordReset(token: string, password: string): Promise<string>;
};

export const accountAccess: AccountAccess = {
  register: (email) => post<RegisterResult>(`${BASE}/api/auth/register`, { email }),

  async resendConfirmation(email) {
    await post<{ ok: true }>(`${BASE}/api/auth/register/resend`, { email });
  },

  /** A POST rather than a GET with the token in the query string: proxies and
   *  hosting dashboards log query strings, and this token is a working key
   *  until it is spent. */
  emailLink: (kind, token) => post<EmailLink>(`${BASE}/api/auth/links/check`, { kind, token }),

  async completeRegistration(token, password) {
    const { email } = await post<SignedIn>(`${BASE}/api/auth/register/complete`, { token, password });
    return email;
  },

  async requestPasswordReset(email) {
    const sent = await post<{ ok: true; email: string }>(`${BASE}/api/auth/password/forgot`, { email });
    return sent.email;
  },

  async completePasswordReset(token, password) {
    const { email } = await post<SignedIn>(`${BASE}/api/auth/password/reset`, { token, password });
    return email;
  },
};

export const api = {
  /** Whether this browser is signed in, and as whom — asked before anything
   *  else, so the app shows the right door rather than a chat surface whose
   *  every request fails. */
  async session(): Promise<Session> {
    const res = await request(`${BASE}/api/auth/session`, { cache: "no-store" });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    return res.json() as Promise<Session>;
  },

  /** Exchanges an email and password for a session cookie. The cookie is
   *  HttpOnly, so nothing here ever sees it — the browser holds it and sends it
   *  back on every request. */
  async signIn(email: string, password: string): Promise<string> {
    const signedIn = await post<SignedIn>(`${BASE}/api/auth/login`, { email, password });
    return signedIn.email;
  },

  /** Changes the password. On the server this ends every session including this
   *  browser's, which is then handed a fresh one in the same response — so it
   *  resolves with the owner still signed in, holding a different cookie, and
   *  every other device signed out. */
  async changePassword(current: string, next: string): Promise<void> {
    await post<SignedIn>(`${BASE}/api/auth/password`, { current, next });
  },

  /** Ends this session, and optionally every other one.
   *
   *  Throws when the server could not do it. That matters most for
   *  `everywhere`: it is the button somebody presses because they think a
   *  device is in the wrong hands, and a silent failure would tell them the
   *  danger had passed while it had not. */
  async signOut(everywhere = false): Promise<void> {
    await post<{ ok: true }>(`${BASE}/api/auth/logout`, { everywhere });
  },

  async providers(): Promise<Provider[]> {
    return json(await request(`${BASE}/api/providers`, { cache: "no-store" }));
  },

  /** Every conversation, archived included — the archive is a filter, not a
   *  separate store. A query searches titles, folders and message bodies in
   *  Postgres and returns an excerpt for a body match. */
  async conversations(search = ""): Promise<Conversation[]> {
    const q = search.trim() ? `?q=${encodeURIComponent(search.trim())}` : "";
    return json(await request(`${BASE}/api/conversations${q}`, { cache: "no-store" }));
  },

  async conversation(id: string): Promise<Conversation> {
    return json(await request(`${BASE}/api/conversations/${id}`, { cache: "no-store" }));
  },

  async create(input: {
    folder?: string;
    provider?: ProviderId;
    model?: Model;
  }): Promise<Conversation> {
    return json(
      await request(`${BASE}/api/conversations`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      }),
    );
  },

  async patch(
    id: string,
    patch: { title?: string; archived?: boolean; folder?: string },
  ): Promise<Conversation> {
    return json(
      await request(`${BASE}/api/conversations/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      }),
    );
  },

  /** What is inside a directory ON THE OWNER'S MACHINE. The server forwards
   *  this to the daemon and never touches a filesystem itself — it is meant to
   *  run somewhere else entirely. An empty path asks for the starting points. */
  async directories(path: string): Promise<DirListing> {
    const q = path ? `?path=${encodeURIComponent(path)}` : "";
    return json(await request(`${BASE}/api/directories${q}`, { cache: "no-store" }));
  },

  /** Folders already in use, most recent first. Free from conversations that
   *  already exist, so there is no separate list to maintain. */
  async recentFolders(): Promise<string[]> {
    const body = await json<{ folders: string[] }>(
      await request(`${BASE}/api/folders/recent`, { cache: "no-store" }),
    );
    return body.folders;
  },

  async remove(id: string): Promise<void> {
    const res = await request(`${BASE}/api/conversations/${id}`, { method: "DELETE" });
    // 204, so there is no body to decode — but a 401 still has to become the
    // same NotSignedIn every other call throws, or deleting would be the one
    // action that reports "401" instead of showing the login screen.
    if (res.status === 401) throw new NotSignedIn();
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  },

  /** What moving to a provider would cost. Charges nothing — that is the point:
   *  the number has to be available before the decision, not after it. */
  async switchCost(
    id: string,
    provider: ProviderId,
  ): Promise<{ messagesToReplay: number; estimatedTokens: number }> {
    return json(
      await request(
        `${BASE}/api/conversations/${id}/switch-cost?provider=${encodeURIComponent(provider)}`,
        { cache: "no-store" },
      ),
    );
  },

  /** Ends a turn that is already running. Resolves to false when the turn had
   *  already finished — the click and the last delta race every time, and that
   *  is an ordinary outcome rather than something worth a message. The turn's
   *  actual ending still arrives over the socket, like every other ending. */
  async stopTurn(turnId: string): Promise<boolean> {
    const res = await request(`${BASE}/api/turns/${turnId}/stop`, { method: "POST" });
    if (res.status === 409) return false;
    if (res.status === 401) throw new NotSignedIn();
    if (!res.ok) {
      const body = (await res.json().catch(() => null)) as { error?: string } | null;
      throw new Error(body?.error ?? `${res.status} ${res.statusText}`);
    }
    return true;
  },

  async send(
    id: string,
    input: { text: string; provider: ProviderId; model: Model },
  ): Promise<{ turnId: string; entry: Entry }> {
    return json(
      await request(`${BASE}/api/conversations/${id}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      }),
    );
  },
};

// ---------------------------------------------------------------------------

export type ServerEvent =
  | { type: "providers"; providers: Provider[] }
  | { type: "daemon"; online: boolean }
  | { type: "conversation"; conversation: Conversation }
  | { type: "entry_added"; conversationId: string; entry: Entry }
  | { type: "entry_delta"; conversationId: string; entryId: string; text: string }
  // Only ever an agent turn: entry_done is what closes one, and the user's own
  // messages and replay markers are complete the moment they are written.
  | { type: "entry_done"; conversationId: string; entry: AgentMessage };

/** Opens the live feed and keeps it open.
 *
 *  Reconnect is bounded exponential backoff with full jitter, and the attempt
 *  counter resets only once a socket has actually opened — resetting when the
 *  dial returns is what turns a flapping connection into a hot loop. Returns a
 *  teardown that stops retrying as well as closing. */
export function connect(
  onEvent: (ev: ServerEvent) => void,
  onOnline: (online: boolean) => void,
): () => void {
  const url = BASE.replace(/^http/, "ws") + "/ws";
  let socket: WebSocket | null = null;
  let attempt = 0;
  let stopped = false;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const open = () => {
    if (stopped) return;
    socket = new WebSocket(url);

    socket.onopen = () => {
      attempt = 0;
      onOnline(true);
    };
    socket.onmessage = (e) => {
      try {
        onEvent(JSON.parse(e.data as string) as ServerEvent);
      } catch {
        // A malformed frame is not worth tearing the connection down for.
      }
    };
    socket.onclose = () => {
      onOnline(false);
      if (stopped) return;
      const ceiling = Math.min(500 * 2 ** attempt, 30_000);
      attempt += 1;
      timer = setTimeout(open, Math.random() * ceiling);
    };
    socket.onerror = () => socket?.close();
  };

  open();

  return () => {
    stopped = true;
    if (timer) clearTimeout(timer);
    socket?.close();
  };
}
