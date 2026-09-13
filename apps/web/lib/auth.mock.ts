import { toast } from "sonner";
import type { AccountAccess, EmailLink, EmailLinkKind, RegisterResult } from "./api";

/* MOCK — account access (US1) until the Go endpoints exist.
 *
 * Implements exactly the contract in api.ts (`AccountAccess`) so the screens can
 * be confirmed in the real app before any backend is built. Swapping it out is
 * one import in queries.ts; nothing else should ever import this file except
 * for MOCK_LANDING_NOTE, which goes with it.
 *
 * Nothing here talks to the server. State lives in localStorage so an "email"
 * link opened in a new tab still finds its token. Emails are toasts with an
 * "Open link" action.
 *
 * Seeded so every path is reachable:
 *   invited, no account   agent@sparstrow.com, priya.nair@northwindlogistics.com
 *   invited, has account  marcus.bell@sparstrow.com   (also the only resettable one)
 *   anything else         becomes an access request
 *   an address containing +maildown   "the email could not be sent"
 *   an address containing +throttle   "too many attempts"
 *   /verify?token=offline             the link check cannot reach the server
 *
 * Deliberately NOT a real session: the mock cannot sign anybody in to the Go
 * API, so finishing registration or a reset returns to sign in with a note
 * instead of landing in Chat. The real endpoint sets the session cookie.
 */

export const MOCK_LANDING_NOTE =
  "Mock data: the real backend signs you in straight away. Nothing was created on the server.";

const STORE = "sparstrowgen.account-access.mock";
const LINK_LIFETIME_MS = 30 * 60_000;
const RESEND_GAP_MS = 30_000;
const MIN_PASSWORD = 12;
const EMAIL = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;

const INVITED = [
  "agent@sparstrow.com",
  "priya.nair@northwindlogistics.com",
  "marcus.bell@sparstrow.com",
];

type Token = {
  kind: EmailLinkKind;
  email: string;
  issuedAt: number;
  used: boolean;
  superseded: boolean;
};

type State = {
  accounts: string[];
  requests: Record<string, number>;
  tokens: Record<string, Token>;
  lastSentAt: Record<string, number>;
};

const fresh = (): State => ({
  accounts: ["marcus.bell@sparstrow.com"],
  requests: {},
  tokens: {},
  lastSentAt: {},
});

function load(): State {
  const raw = localStorage.getItem(STORE);
  return raw ? (JSON.parse(raw) as State) : fresh();
}

const save = (s: State) => localStorage.setItem(STORE, JSON.stringify(s));
const norm = (email: string) => email.trim().toLowerCase();
const pause = () => new Promise((r) => setTimeout(r, 600));

function validate(email: string) {
  if (!EMAIL.test(email)) throw new Error("that does not look like an email address");
  if (email.includes("+throttle")) throw new Error("too many attempts — try again in 8s");
  if (email.includes("+maildown")) {
    throw new Error("the email could not be sent just now — try again in a few minutes");
  }
}

function deliver(to: string, subject: string, link?: string) {
  toast(`Mock email to ${to}`, {
    description: subject,
    duration: 60_000,
    action: link ? { label: "Open link", onClick: () => window.location.assign(link) } : undefined,
  });
}

function issue(s: State, kind: EmailLinkKind, email: string): string {
  for (const t of Object.values(s.tokens)) {
    if (t.kind === kind && t.email === email && !t.used) t.superseded = true;
  }
  const id = crypto.randomUUID();
  s.tokens[id] = { kind, email, issuedAt: Date.now(), used: false, superseded: false };
  return `${window.location.origin}/${kind}?token=${id}`;
}

function inspect(s: State, kind: EmailLinkKind, token: string): EmailLink {
  const t = s.tokens[token];
  if (!t || t.kind !== kind) return { usable: false, reason: "expired", accountReady: false };
  const accountReady = s.accounts.includes(t.email);
  if (t.used) return { usable: false, reason: "used", accountReady };
  if (t.superseded) return { usable: false, reason: "superseded", accountReady };
  if (Date.now() - t.issuedAt > LINK_LIFETIME_MS) {
    return { usable: false, reason: "expired", accountReady };
  }
  return { usable: true, email: t.email };
}

function sendConfirmation(s: State, email: string) {
  s.lastSentAt[email] = Date.now();
  if (s.accounts.includes(email)) {
    deliver(email, "You already have a sparstrowgen account", `${window.location.origin}/`);
  } else {
    deliver(email, "Finish creating your sparstrowgen account", issue(s, "verify", email));
  }
}

function consume(kind: EmailLinkKind, token: string, password: string): [State, Token] {
  const s = load();
  if (!inspect(s, kind, token).usable) {
    throw new Error("this link no longer works — open the newest email or ask for another");
  }
  if (password.length < MIN_PASSWORD) {
    throw new Error(`a password needs to be at least ${MIN_PASSWORD} characters`);
  }
  const t = s.tokens[token];
  t.used = true;
  return [s, t];
}

export const accountAccessMock: AccountAccess = {
  async register(raw): Promise<RegisterResult> {
    await pause();
    const email = norm(raw);
    validate(email);
    const s = load();
    if (!INVITED.includes(email)) {
      if (!s.requests[email]) deliver("the owner", `Access request from ${email}`);
      s.requests[email] = (s.requests[email] ?? 0) + 1;
      save(s);
      return { outcome: "request-sent", email };
    }
    sendConfirmation(s, email);
    save(s);
    return { outcome: "check-email", email };
  },

  async resendConfirmation(raw) {
    await pause();
    const email = norm(raw);
    validate(email);
    const s = load();
    if (Date.now() - (s.lastSentAt[email] ?? 0) < RESEND_GAP_MS) {
      throw new Error("wait a moment before sending another");
    }
    if (INVITED.includes(email)) sendConfirmation(s, email);
    save(s);
  },

  async emailLink(kind, token) {
    await pause();
    if (token === "offline") throw new Error("Failed to fetch");
    return inspect(load(), kind, token);
  },

  async completeRegistration(token, password) {
    await pause();
    const [s, t] = consume("verify", token, password);
    if (!s.accounts.includes(t.email)) s.accounts.push(t.email);
    save(s);
    return t.email;
  },

  async requestPasswordReset(raw) {
    await pause();
    const email = norm(raw);
    validate(email);
    const s = load();
    if (s.accounts.includes(email)) {
      deliver(email, "Reset your sparstrowgen password", issue(s, "reset", email));
      save(s);
    }
    return email;
  },

  async completePasswordReset(token, password) {
    await pause();
    const [s, t] = consume("reset", token, password);
    deliver(t.email, "Your sparstrowgen password was changed");
    save(s);
    return t.email;
  },
};
