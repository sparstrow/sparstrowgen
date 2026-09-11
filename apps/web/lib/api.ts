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

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) {
    // The server sends {"error": "..."} and its wording is written to be read
    // by the owner, so it is surfaced rather than replaced with a generic one.
    const body = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new Error(body?.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export const api = {
  async providers(): Promise<Provider[]> {
    return json(await fetch(`${BASE}/api/providers`, { cache: "no-store" }));
  },

  /** Every conversation, archived included — the archive is a filter, not a
   *  separate store. A query searches titles, folders and message bodies in
   *  Postgres and returns an excerpt for a body match. */
  async conversations(search = ""): Promise<Conversation[]> {
    const q = search.trim() ? `?q=${encodeURIComponent(search.trim())}` : "";
    return json(await fetch(`${BASE}/api/conversations${q}`, { cache: "no-store" }));
  },

  async conversation(id: string): Promise<Conversation> {
    return json(await fetch(`${BASE}/api/conversations/${id}`, { cache: "no-store" }));
  },

  async create(input: {
    folder?: string;
    provider?: ProviderId;
    model?: Model;
  }): Promise<Conversation> {
    return json(
      await fetch(`${BASE}/api/conversations`, {
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
      await fetch(`${BASE}/api/conversations/${id}`, {
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
    return json(await fetch(`${BASE}/api/directories${q}`, { cache: "no-store" }));
  },

  /** Folders already in use, most recent first. Free from conversations that
   *  already exist, so there is no separate list to maintain. */
  async recentFolders(): Promise<string[]> {
    const body = await json<{ folders: string[] }>(
      await fetch(`${BASE}/api/folders/recent`, { cache: "no-store" }),
    );
    return body.folders;
  },

  async remove(id: string): Promise<void> {
    const res = await fetch(`${BASE}/api/conversations/${id}`, { method: "DELETE" });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  },

  /** What moving to a provider would cost. Charges nothing — that is the point:
   *  the number has to be available before the decision, not after it. */
  async switchCost(
    id: string,
    provider: ProviderId,
  ): Promise<{ messagesToReplay: number; estimatedTokens: number }> {
    return json(
      await fetch(
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
    const res = await fetch(`${BASE}/api/turns/${turnId}/stop`, { method: "POST" });
    if (res.status === 409) return false;
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
      await fetch(`${BASE}/api/conversations/${id}/messages`, {
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
