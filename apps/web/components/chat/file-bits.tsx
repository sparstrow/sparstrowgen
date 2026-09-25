"use client";

import {
  File,
  FileCode,
  FileSpreadsheet,
  FileText,
  Film,
  Image as ImageIcon,
  Music,
  RotateCw,
  Sparkles,
  X,
} from "lucide-react";
import { cn } from "cn";
import { api } from "@/lib/api";
import type { ConversationFile } from "@/lib/chat-types";
import { fileKind, formatBytes, useFilesPane, type Attachment, type FileKind } from "@/lib/files";

/* The pieces a file is shown as: in a message, under an answer, waiting in the
   message box, and as a row in the pane. Every one of them opens the file in
   the pane's preview. */

export function FileKindIcon({ kind, className }: { kind: FileKind; className?: string }) {
  const Icon =
    kind === "image"
      ? ImageIcon
      : kind === "csv"
        ? FileSpreadsheet
        : kind === "text" || kind === "markdown"
          ? FileCode
          : kind === "pdf"
            ? FileText
            : kind === "audio"
              ? Music
              : kind === "video"
                ? Film
                : File;
  return <Icon className={className} aria-hidden />;
}

function extension(name: string) {
  return name.includes(".") ? name.split(".").pop()!.toUpperCase() : "FILE";
}

/** A picture in the transcript: the picture itself, since that is what was
 *  sent. An agent's output is shown larger, with its name, because it is the
 *  answer rather than a reference. */
function FileTile({ file, large }: { file: ConversationFile; large?: boolean }) {
  const show = useFilesPane((s) => s.show);
  return (
    <button
      type="button"
      onClick={() => show({ source: "chat", fileId: file.id })}
      aria-label={`Open ${file.name}`}
      className={cn(
        "block cursor-zoom-in overflow-hidden rounded-lg border bg-muted text-left transition-colors hover:border-ring",
        large && "max-w-lg",
      )}
    >
      {/* eslint-disable-next-line @next/next/no-img-element -- bytes from our API, sized by the file itself */}
      <img
        src={api.fileUrl(file.id)}
        alt={file.name}
        loading="lazy"
        // Whole, never cropped: a screenshot's point is often at its edge.
        className={cn("block object-contain", large ? "max-h-96 w-full" : "h-28 w-auto max-w-64")}
      />
      {large && (
        <span className="flex items-center gap-1.5 border-t bg-card px-2.5 py-1.5 text-xs text-muted-foreground">
          <Sparkles className="size-3 shrink-0" aria-hidden />
          <span className="truncate">{file.name}</span>
          <span aria-hidden>·</span>
          <span className="shrink-0">{formatBytes(file.size)}</span>
        </span>
      )}
    </button>
  );
}

/** Any other file in the transcript: what it is and how big. */
function FileCard({ file }: { file: ConversationFile }) {
  const show = useFilesPane((s) => s.show);
  const kind = fileKind(file.name, file.mediaType);
  return (
    <button
      type="button"
      onClick={() => show({ source: "chat", fileId: file.id })}
      className="flex max-w-72 min-w-0 items-center gap-2.5 rounded-lg border bg-card py-2 pr-3 pl-2.5 text-left transition-colors hover:border-ring"
    >
      <span className="grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
        <FileKindIcon kind={kind} className="size-4" />
      </span>
      <span className="min-w-0">
        <span className="block truncate text-[13px] font-medium">{file.name}</span>
        <span className="block text-xs text-muted-foreground">
          {formatBytes(file.size)} · {extension(file.name)}
        </span>
      </span>
    </button>
  );
}

/** The files of one message or one answer. */
export function FileList({
  files,
  align,
  outputs,
}: {
  files: ConversationFile[];
  align: "start" | "end";
  /** An agent's answer: its pictures are shown large. */
  outputs?: boolean;
}) {
  return (
    <div className={cn("flex flex-wrap gap-2", align === "end" ? "justify-end" : "justify-start")}>
      {files.map((f) =>
        fileKind(f.name, f.mediaType) === "image" ? (
          <FileTile key={f.id} file={f} large={outputs} />
        ) : (
          <FileCard key={f.id} file={f} />
        ),
      )}
    </div>
  );
}

/** One file waiting in the message box: uploading, up, or failed. */
export function AttachmentChip({
  attachment,
  onRemove,
  onRetry,
}: {
  attachment: Attachment;
  onRemove: () => void;
  onRetry: () => void;
}) {
  const a = attachment;
  const kind = fileKind(a.name, a.file?.mediaType);
  return (
    <span
      className={cn(
        "relative flex h-11 max-w-60 items-center gap-2 rounded-lg border bg-card py-1 pr-8 pl-1 text-xs",
        a.status === "failed" && "border-destructive/50",
      )}
    >
      {a.previewUrl ? (
        <span
          className="size-[34px] shrink-0 rounded-md bg-muted bg-cover bg-center"
          style={{ backgroundImage: `url(${a.previewUrl})` }}
          aria-hidden
        />
      ) : (
        <span className="grid size-[34px] shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
          <FileKindIcon kind={kind} className="size-4" />
        </span>
      )}
      <span className="min-w-0 leading-4">
        <span className="block truncate font-medium">{a.name}</span>
        {a.status === "uploading" ? (
          <span className="text-muted-foreground">
            Uploading… {Math.round(a.progress * 100)}%
          </span>
        ) : a.status === "failed" ? (
          <button
            type="button"
            onClick={onRetry}
            title={a.error}
            className="flex items-center gap-1 text-destructive-text hover:underline"
          >
            Didn’t upload · Try again
            <RotateCw className="size-3" aria-hidden />
          </button>
        ) : (
          <span className="text-muted-foreground">{formatBytes(a.size)}</span>
        )}
      </span>
      {a.status === "uploading" && (
        <span className="absolute inset-x-1 bottom-0.5 h-0.5 overflow-hidden rounded-full bg-muted" aria-hidden>
          <span className="block h-full bg-foreground transition-[width]" style={{ width: `${a.progress * 100}%` }} />
        </span>
      )}
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove ${a.name}`}
        className="absolute top-1 right-1 grid size-[22px] place-items-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X className="size-3.5" />
      </button>
    </span>
  );
}
