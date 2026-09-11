/* A conversation with no name.

`title` is empty until a conversation has one — from the first thing said in it,
or from the owner typing one. That emptiness used to be a string in the database
reading "Untitled conversation", which meant every surface treated a description
as though it were a name: search matched it, and nothing could tell a
conversation nobody had named from one named that.

The placeholder lives here so the two places a conversation is named — the
sidebar row and the header above the transcript — cannot drift apart, and so
that it stays visibly a description: muted, where a real name is not. */

export const unnamedConversation = "Untitled conversation";

/** The name as prose, for somewhere that needs a plain string — an aria-label,
 *  a sentence in a dialog. */
export function conversationName(title: string) {
  return title || unnamedConversation;
}

export function ConversationName({
  title,
  className = "",
}: {
  title: string;
  className?: string;
}) {
  return (
    <span className={`${className} ${title ? "" : "text-muted-foreground"}`}>
      {conversationName(title)}
    </span>
  );
}
