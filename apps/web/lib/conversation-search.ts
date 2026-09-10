import type { Conversation, SearchHit } from "./chat-types";

/* Searching titles alone would miss the conversations that most need finding —
   several are literally called "Untitled conversation", and a conversation only
   gets a good title if someone bothered to rename it. So message text counts,
   and the matching line is shown so a hit in a 200-message transcript is
   explicable rather than mysterious.

   This runs over everything in memory, which is honest for a prototype and
   wrong for real: the transcripts live in Postgres and the search belongs
   there. See docs/KnownGaps.md G-7. */

const EXCERPT_PAD = 34;

function excerptAround(text: string, at: number, term: number): string {
  const start = Math.max(0, at - EXCERPT_PAD);
  const end = Math.min(text.length, at + term + EXCERPT_PAD);
  return (
    (start > 0 ? "…" : "") +
    text.slice(start, end).trim() +
    (end < text.length ? "…" : "")
  );
}

export type SearchResult = {
  conversation: Conversation;
  hit: SearchHit;
};

export function searchConversations(
  conversations: Conversation[],
  query: string,
): SearchResult[] {
  const q = query.trim().toLowerCase();
  if (!q) return conversations.map((c) => ({ conversation: c, hit: { inTitle: false } }));

  const results: SearchResult[] = [];

  for (const c of conversations) {
    const inTitle = c.title.toLowerCase().includes(q);
    const inFolder = c.folder.toLowerCase().includes(q);

    let excerpt: string | undefined;
    for (const e of c.entries) {
      if (e.role === "replay") continue;
      const at = e.text.toLowerCase().indexOf(q);
      if (at !== -1) {
        excerpt = excerptAround(e.text, at, q.length);
        break;
      }
    }

    if (inTitle || inFolder || excerpt) {
      // A title match needs no excerpt — the reason for the hit is already on
      // screen, and showing one anyway just adds a line that explains nothing.
      results.push({
        conversation: c,
        hit: { inTitle, excerpt: inTitle ? undefined : excerpt },
      });
    }
  }

  return results;
}
