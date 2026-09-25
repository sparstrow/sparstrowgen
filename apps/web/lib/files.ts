"use client";

import { create } from "zustand";
import { api } from "./api";
import type { ConversationFile } from "./chat-types";

/* Files in a conversation (docs/Decisions.md D-056), the browser's side.
 *
 * Two pieces of VIEW state live here, beside the draft text they belong with:
 * the files waiting in a conversation's message box, and what the files pane
 * is showing. Neither is the server's record of a file — that is the Query
 * cache's (lib/queries.ts, keys.files) — so nothing here is patched by events. */

/** The most a conversation keeps per file. Mirrors protocol.MaxFileBytes: a
 *  file over it is refused the moment it is added, not after uploading it. */
export const MAX_FILE_BYTES = 25 * 1024 * 1024;

export type Attachment = {
  /** This box's own key for it, before and after the server has it. */
  key: string;
  name: string;
  size: number;
  /** Shown in the chip before the server has seen the file. Revoked when the
   *  chip goes. */
  previewUrl?: string;
  status: "uploading" | "ready" | "failed";
  progress: number;
  /** Set once uploaded; this is what is sent with the message. */
  file?: ConversationFile;
  error?: string;
};

type Held = { file: File; controller: AbortController };

// The File objects and their uploads are not state anyone renders, so they are
// kept out of the store: a File is not serialisable, and an AbortController
// changing would re-render every chip.
const held = new Map<string, Held>();

let counter = 0;

type AttachmentsState = {
  /** Per conversation, in the order added. */
  byConversation: Record<string, Attachment[]>;
  /** Files refused when added (too large), said once under the box. */
  refused: Record<string, { name: string; size: number }[]>;
  add: (conversationId: string, files: File[]) => void;
  retry: (conversationId: string, key: string) => void;
  remove: (conversationId: string, key: string) => void;
  /** After a send: the files now belong to the message. */
  clear: (conversationId: string) => void;
};

const isPictureName = (name: string) => /\.(png|jpe?g|gif|webp)$/i.test(name);

export const useAttachments = create<AttachmentsState>((set, get) => {
  const patch = (conversationId: string, key: string, change: Partial<Attachment>) =>
    set((s) => ({
      byConversation: {
        ...s.byConversation,
        [conversationId]: (s.byConversation[conversationId] ?? []).map((a) =>
          a.key === key ? { ...a, ...change } : a,
        ),
      },
    }));

  const upload = (conversationId: string, key: string) => {
    const h = held.get(key);
    if (!h) return;
    patch(conversationId, key, { status: "uploading", progress: 0, error: undefined });
    api
      .uploadFile(
        conversationId,
        h.file,
        (progress) => patch(conversationId, key, { progress }),
        h.controller.signal,
      )
      .then((file) => patch(conversationId, key, { status: "ready", progress: 1, file }))
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;
        patch(conversationId, key, {
          status: "failed",
          error: error instanceof Error ? error.message : "It did not upload.",
        });
      });
  };

  return {
    byConversation: {},
    refused: {},

    add: (conversationId, files) => {
      const accepted: Attachment[] = [];
      const refused: { name: string; size: number }[] = [];
      for (const file of files) {
        if (file.size > MAX_FILE_BYTES) {
          refused.push({ name: file.name, size: file.size });
          continue;
        }
        const key = `a${++counter}`;
        held.set(key, { file, controller: new AbortController() });
        accepted.push({
          key,
          name: file.name,
          size: file.size,
          previewUrl: isPictureName(file.name) ? URL.createObjectURL(file) : undefined,
          status: "uploading",
          progress: 0,
        });
      }
      set((s) => ({
        byConversation: {
          ...s.byConversation,
          [conversationId]: [...(s.byConversation[conversationId] ?? []), ...accepted],
        },
        refused: { ...s.refused, [conversationId]: refused },
      }));
      for (const a of accepted) upload(conversationId, a.key);
    },

    retry: (conversationId, key) => {
      const h = held.get(key);
      if (!h) return;
      held.set(key, { file: h.file, controller: new AbortController() });
      upload(conversationId, key);
    },

    remove: (conversationId, key) => {
      const a = (get().byConversation[conversationId] ?? []).find((x) => x.key === key);
      held.get(key)?.controller.abort();
      held.delete(key);
      if (a?.previewUrl) URL.revokeObjectURL(a.previewUrl);
      // Taken back off the server too, so it does not linger as an upload
      // nobody sent. Failing that is harmless: an unsent upload is never
      // listed or given to an agent.
      if (a?.file) void api.removeFile(a.file.id).catch(() => undefined);
      set((s) => ({
        byConversation: {
          ...s.byConversation,
          [conversationId]: (s.byConversation[conversationId] ?? []).filter((x) => x.key !== key),
        },
      }));
    },

    clear: (conversationId) => {
      for (const a of get().byConversation[conversationId] ?? []) {
        held.delete(a.key);
        if (a.previewUrl) URL.revokeObjectURL(a.previewUrl);
      }
      set((s) => {
        const byConversation = { ...s.byConversation };
        delete byConversation[conversationId];
        return { byConversation, refused: { ...s.refused, [conversationId]: [] } };
      });
    },
  };
});

const NONE: Attachment[] = [];
const NONE_REFUSED: { name: string; size: number }[] = [];

/** A stable empty list when there is nothing, so a selector does not hand back
 *  a fresh array on every store write. */
export function useConversationAttachments(conversationId: string | null) {
  return useAttachments((s) => (conversationId ? (s.byConversation[conversationId] ?? NONE) : NONE));
}

export function useRefusedFiles(conversationId: string | null) {
  return useAttachments((s) => (conversationId ? (s.refused[conversationId] ?? NONE_REFUSED) : NONE_REFUSED));
}

/* -------------------------------------------------------------------------
   The files pane. Open or shut, which tab, and what is being previewed: one
   window's choice, like the transcript view. */

export type PaneTab = "chat" | "folder";

/** A file to preview: one of the conversation's own, or one in its working
 *  folder by path. */
export type PreviewTarget =
  | { source: "chat"; fileId: string }
  | { source: "folder"; path: string; size: number };

type FilesPaneState = {
  open: boolean;
  tab: PaneTab;
  /** Where the working-folder tab is, per conversation. */
  folderPath: Record<string, string>;
  preview: PreviewTarget | null;
  wide: boolean;
  toggle: () => void;
  close: () => void;
  setTab: (tab: PaneTab) => void;
  setFolderPath: (conversationId: string, path: string) => void;
  show: (target: PreviewTarget) => void;
  back: () => void;
  toggleWide: () => void;
};

export const useFilesPane = create<FilesPaneState>((set) => ({
  open: false,
  tab: "chat",
  folderPath: {},
  preview: null,
  wide: false,
  toggle: () => set((s) => ({ open: !s.open, preview: null, wide: false })),
  close: () => set({ open: false, preview: null, wide: false }),
  setTab: (tab) => set({ tab, preview: null, wide: false }),
  setFolderPath: (conversationId, path) =>
    set((s) => ({ folderPath: { ...s.folderPath, [conversationId]: path } })),
  show: (preview) =>
    set({ open: true, preview, tab: preview.source === "chat" ? "chat" : "folder" }),
  back: () => set({ preview: null, wide: false }),
  toggleWide: () => set((s) => ({ wide: !s.wide })),
}));

/* -------------------------------------------------------------------------
   What a file is, for choosing how to show it. By its name first, because the
   working folder's files come with nothing else; the server's media type,
   where there is one, settles pictures and PDFs. */

export type FileKind = "image" | "pdf" | "markdown" | "csv" | "text" | "audio" | "video" | "other";

const TEXT_EXT = new Set([
  "txt", "log", "json", "jsonl", "yaml", "yml", "toml", "ini", "cfg", "conf", "env", "xml",
  "ts", "tsx", "js", "jsx", "mjs", "cjs", "go", "py", "rb", "rs", "java", "kt", "cs", "c", "h",
  "cpp", "hpp", "swift", "php", "sql", "sh", "ps1", "bat", "cmd", "css", "scss", "html", "htm",
  "svg", "al", "edi", "x12", "map", "gitignore", "dockerfile", "makefile", "lock", "tsv", "diff", "patch",
]);

export function fileKind(name: string, mediaType?: string): FileKind {
  const ext = name.includes(".") ? name.split(".").pop()!.toLowerCase() : name.toLowerCase();
  if (["png", "jpg", "jpeg", "gif", "webp", "avif", "bmp", "ico"].includes(ext)) return "image";
  if (ext === "pdf" || mediaType === "application/pdf") return "pdf";
  if (ext === "md" || ext === "markdown") return "markdown";
  if (ext === "csv") return "csv";
  if (["mp3", "wav", "ogg", "m4a", "flac"].includes(ext)) return "audio";
  if (["mp4", "webm", "mov"].includes(ext)) return "video";
  if (TEXT_EXT.has(ext) || mediaType?.startsWith("text/") || mediaType === "application/json")
    return "text";
  if (mediaType?.startsWith("image/") && mediaType !== "image/svg+xml") return "image";
  return "other";
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

/** What an agent can do with a file sent to it, said before sending. Mirrors
 *  the daemon's filesPrompt: codex sees pictures and small text files and
 *  nothing else under its sandbox (docs/Capabilities.md). */
export function unreadableBy(provider: string, attachments: Attachment[]): string[] {
  if (provider !== "codex") return [];
  return attachments
    .filter((a) => {
      // Exactly the pictures the daemon hands codex with --image.
      if (/\.(png|jpe?g|gif|webp)$/i.test(a.name)) return false;
      const kind = fileKind(a.name, a.file?.mediaType);
      if ((kind === "text" || kind === "csv" || kind === "markdown") && a.size <= 100 * 1024)
        return false;
      return true;
    })
    .map((a) => a.name);
}
