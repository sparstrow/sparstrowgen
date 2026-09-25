"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ChevronRight,
  Download,
  Folder,
  FolderOpen,
  Maximize2,
  Minimize2,
  PlugZap,
  RefreshCw,
  Sparkles,
  TriangleAlert,
  Upload,
  X,
} from "lucide-react";
import { cn } from "cn";
import { api } from "@/lib/api";
import type { ConversationFile } from "@/lib/chat-types";
import { useConversationFiles, useFolder } from "@/lib/queries";
import { fileKind, formatBytes, useFilesPane, type PaneTab, type PreviewTarget } from "@/lib/files";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ChoiceToggle } from "./choice-toggle";
import { FileKindIcon } from "./file-bits";
import { Markdown } from "./markdown";
import { clockTime } from "./provider-meta";

/* The files pane (docs/design/prototypes/Chat/files.handoff.md): this chat's
   uploads and outputs, which the server keeps, and its working folder, which
   is read from the computer each time it is looked at. Either opens a preview. */

const TABS: { id: PaneTab; label: string }[] = [
  { id: "chat", label: "This chat" },
  { id: "folder", label: "Working folder" },
];

export function FilesPane({
  conversationId,
  folderName,
  reachable,
}: {
  conversationId: string;
  /** The working folder's own name, for the top of its breadcrumbs. */
  folderName: string;
  /** Whether the computer can be asked for the working folder at all. */
  reachable: boolean;
}) {
  const tab = useFilesPane((s) => s.tab);
  const setTab = useFilesPane((s) => s.setTab);
  const close = useFilesPane((s) => s.close);
  const preview = useFilesPane((s) => s.preview);
  const wide = useFilesPane((s) => s.wide);

  return (
    <aside
      aria-label="Files"
      className={cn(
        // Beside the conversation where there is room; over it where there is
        // not, since a 400px column would leave a phone's conversation unreadable.
        "absolute inset-y-0 right-0 z-20 flex w-full flex-col border-l bg-background shadow-xl sm:w-[400px] lg:static lg:z-auto lg:shadow-none",
        wide && preview && "sm:w-[min(56vw,760px)] lg:w-[min(56vw,760px)]",
      )}
    >
      {preview ? (
        <Preview conversationId={conversationId} target={preview} />
      ) : (
        <>
          <div className="flex h-14 shrink-0 items-center gap-2 border-b pr-2 pl-3">
            <ChoiceToggle options={TABS} value={tab} onChange={setTab} label="Files to show" />
            <span className="flex-1" />
            <Button variant="ghost" size="icon" className="size-8" onClick={close} aria-label="Close files">
              <X className="size-4" />
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto">
            {tab === "chat" ? (
              <ChatFiles conversationId={conversationId} />
            ) : (
              <WorkingFolder
                conversationId={conversationId}
                folderName={folderName}
                reachable={reachable}
              />
            )}
          </div>
        </>
      )}
    </aside>
  );
}

// ---------------------------------------------------------------------------
// This chat
// ---------------------------------------------------------------------------

function ChatFiles({ conversationId }: { conversationId: string }) {
  const files = useConversationFiles(conversationId);
  const show = useFilesPane((s) => s.show);

  if (files.isPending) return <RowsSkeleton />;
  if (files.isError)
    return (
      <PaneMessage icon={<TriangleAlert className="size-5" />} title="Couldn’t load this chat’s files">
        {files.error.message}
        <Button variant="outline" size="sm" className="mt-3" onClick={() => void files.refetch()}>
          <RefreshCw className="size-3.5" />
          Try again
        </Button>
      </PaneMessage>
    );

  // An upload still waiting in some message box is not the chat's yet.
  const sent = files.data.files.filter((f) => f.entryId);
  if (sent.length === 0)
    return (
      <PaneMessage icon={<Folder className="size-5" />} title="No files in this chat yet">
        Add pictures and files with + in the message box, or drop them on the conversation.
        Pictures an agent makes for you land here too.
      </PaneMessage>
    );

  const outputs = sent.filter((f) => f.origin === "output");
  const uploads = sent.filter((f) => f.origin === "upload");
  const folder = files.data.folder;
  const sep = folder.includes("\\") ? "\\" : "/";
  return (
    <div className="pb-4">
      {outputs.length > 0 && (
        <Section
          icon={<Sparkles className="size-3" />}
          title={`Outputs · ${outputs.length}`}
          path={folder && `${folder}${sep}outputs`}
        >
          {outputs.map((f) => (
            <FileRow key={f.id} file={f} onOpen={() => show({ source: "chat", fileId: f.id })} />
          ))}
        </Section>
      )}
      {uploads.length > 0 && (
        <Section
          icon={<Upload className="size-3" />}
          title={`Uploads · ${uploads.length}`}
          path={folder && `${folder}${sep}uploads`}
        >
          {uploads.map((f) => (
            <FileRow key={f.id} file={f} onOpen={() => show({ source: "chat", fileId: f.id })} />
          ))}
        </Section>
      )}
      <p className="px-4 pt-2 text-xs text-muted-foreground">
        Files an agent creates or changes in your project stay where it put them. Find them under
        Working folder.
      </p>
    </div>
  );
}

function Section({
  icon,
  title,
  path,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  path?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="px-3 pt-3">
      <h3 className="mb-1 flex items-center gap-1.5 px-1 text-xs font-medium text-muted-foreground">
        {icon}
        <span className="shrink-0 whitespace-nowrap">{title}</span>
        {path && (
          // Where it is on the computer, so a file can be found, opened
          // elsewhere or removed there. The end of the path is what tells two
          // conversations apart, so that is the end kept when it is cut.
          <span
            title={path}
            className="ml-auto min-w-0 truncate font-mono text-[11px] font-normal [direction:rtl]"
          >
            {path}
          </span>
        )}
      </h3>
      {children}
    </section>
  );
}

function FileRow({ file, onOpen }: { file: ConversationFile; onOpen: () => void }) {
  const kind = fileKind(file.name, file.mediaType);
  return (
    <button
      type="button"
      onClick={onOpen}
      className="flex w-full items-center gap-2.5 rounded-md p-1.5 text-left hover:bg-accent"
    >
      {kind === "image" ? (
        // eslint-disable-next-line @next/next/no-img-element -- a thumbnail of our own API's bytes
        <img
          src={api.fileUrl(file.id)}
          alt=""
          loading="lazy"
          className="size-9 shrink-0 rounded-md bg-muted object-cover"
        />
      ) : (
        <span className="grid size-9 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
          <FileKindIcon kind={kind} className="size-4" />
        </span>
      )}
      <span className="min-w-0 flex-1 leading-[17px]">
        <span className="block truncate text-[13px] font-medium">{file.name}</span>
        <span className="block truncate text-xs text-muted-foreground">
          {formatBytes(file.size)} · {file.origin === "output" ? "made by the agent" : "you"} ·{" "}
          {clockTime(file.createdAt)}
        </span>
      </span>
    </button>
  );
}

// ---------------------------------------------------------------------------
// Working folder
// ---------------------------------------------------------------------------

function WorkingFolder({
  conversationId,
  folderName,
  reachable,
}: {
  conversationId: string;
  folderName: string;
  reachable: boolean;
}) {
  const path = useFilesPane((s) => s.folderPath[conversationId] ?? "");
  const setPath = useFilesPane((s) => s.setFolderPath);
  const setTab = useFilesPane((s) => s.setTab);
  const show = useFilesPane((s) => s.show);
  const listing = useFolder(conversationId, path, reachable);

  if (!reachable)
    return (
      <PaneMessage icon={<PlugZap className="size-5" />} title="Your computer is unreachable">
        The working folder is read from your computer, so it can’t be shown until it’s back. This
        chat’s own files are still here.
        <Button variant="outline" size="sm" className="mt-3" onClick={() => setTab("chat")}>
          Show this chat’s files
        </Button>
      </PaneMessage>
    );

  const parts = path ? path.split("/") : [];
  const crumbs = (
    <nav aria-label="Folder" className="flex flex-wrap items-center gap-0.5 px-3 pt-2.5 pb-1 text-xs text-muted-foreground">
      {parts.length === 0 ? (
        <span className="px-1 text-foreground">{folderName}</span>
      ) : (
        <button type="button" className="rounded px-1 hover:bg-accent hover:text-foreground" onClick={() => setPath(conversationId, "")}>
          {folderName}
        </button>
      )}
      {parts.map((p, i) => {
        const to = parts.slice(0, i + 1).join("/");
        return (
          <span key={to} className="flex items-center gap-0.5">
            <ChevronRight className="size-3" aria-hidden />
            {i === parts.length - 1 ? (
              <span className="px-1 text-foreground">{p}</span>
            ) : (
              <button type="button" className="rounded px-1 hover:bg-accent hover:text-foreground" onClick={() => setPath(conversationId, to)}>
                {p}
              </button>
            )}
          </span>
        );
      })}
      <Button
        variant="ghost"
        size="icon"
        className="ml-auto size-6"
        onClick={() => void listing.refetch()}
        aria-label="Refresh"
        title="Refresh"
      >
        <RefreshCw className={cn("size-3", listing.isFetching && "animate-spin")} />
      </Button>
    </nav>
  );

  if (listing.isPending) return (<>{crumbs}<RowsSkeleton compact /></>);
  if (listing.isError || listing.data.error)
    return (
      <>
        {crumbs}
        <PaneMessage icon={<TriangleAlert className="size-5" />} title="Couldn’t show this folder">
          {listing.isError ? listing.error.message : listing.data.error}
          <Button variant="outline" size="sm" className="mt-3" onClick={() => void listing.refetch()}>
            <RefreshCw className="size-3.5" />
            Try again
          </Button>
        </PaneMessage>
      </>
    );

  const entries = listing.data.entries;
  if (entries.length === 0)
    return (
      <>
        {crumbs}
        <PaneMessage icon={<FolderOpen className="size-5" />} title="This folder is empty" />
      </>
    );

  return (
    <>
      {crumbs}
      <ul className="px-3 pb-4">
        {entries.map((e) => {
          const full = path ? `${path}/${e.name}` : e.name;
          return (
            <li key={e.name}>
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-left text-[13px] hover:bg-accent"
                onClick={() =>
                  e.kind === "dir"
                    ? setPath(conversationId, full)
                    : show({ source: "folder", path: full, size: e.size })
                }
              >
                {e.kind === "dir" ? (
                  <Folder className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                ) : (
                  <FileKindIcon kind={fileKind(e.name)} className="size-4 shrink-0 text-muted-foreground" />
                )}
                <span className="min-w-0 flex-1 truncate">{e.name}</span>
                {e.kind === "dir" ? (
                  <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
                ) : (
                  <span className="shrink-0 font-mono text-xs text-muted-foreground">{formatBytes(e.size)}</span>
                )}
              </button>
            </li>
          );
        })}
      </ul>
    </>
  );
}

// ---------------------------------------------------------------------------
// Preview
// ---------------------------------------------------------------------------

/** The most text shown at once. A log can be the full 5 MB, and rendering that
 *  as one block of lines freezes the page. */
const TEXT_LIMIT = 512 * 1024;

function Preview({ conversationId, target }: { conversationId: string; target: PreviewTarget }) {
  const back = useFilesPane((s) => s.back);
  const close = useFilesPane((s) => s.close);
  const wide = useFilesPane((s) => s.wide);
  const toggleWide = useFilesPane((s) => s.toggleWide);
  const files = useConversationFiles(conversationId, target.source === "chat");

  const chatFile =
    target.source === "chat" ? files.data?.files.find((f) => f.id === target.fileId) : undefined;
  const name =
    target.source === "chat" ? (chatFile?.name ?? "") : (target.path.split("/").pop() ?? target.path);
  const size = target.source === "chat" ? (chatFile?.size ?? 0) : target.size;
  const url =
    target.source === "chat"
      ? api.fileUrl(target.fileId)
      : api.folderFileUrl(conversationId, target.path);
  const downloadUrl =
    target.source === "chat"
      ? api.fileUrl(target.fileId, true)
      : api.folderFileUrl(conversationId, target.path, true);
  const where =
    target.source === "chat"
      ? chatFile
        ? chatFile.origin === "output"
          ? `Made by the agent · ${clockTime(chatFile.createdAt)}`
          : `Uploaded by you · ${clockTime(chatFile.createdAt)}`
        : ""
      : `Working folder · ${target.path}`;

  return (
    <>
      <div className="flex h-14 shrink-0 items-center gap-1 border-b px-2">
        <Button variant="ghost" size="icon" className="size-8" onClick={back} aria-label="Back to files">
          <ArrowLeft className="size-4" />
        </Button>
        <span className="min-w-0 flex-1 leading-4">
          <span className="block truncate text-[13px] font-medium">{name || "Loading…"}</span>
          <span className="block truncate text-xs text-muted-foreground">
            {size > 0 && `${formatBytes(size)} · `}
            {where}
          </span>
        </span>
        <Button
          variant="ghost"
          size="icon"
          className="hidden size-8 sm:inline-flex"
          onClick={toggleWide}
          aria-pressed={wide}
          aria-label={wide ? "Narrower" : "Wider"}
          title={wide ? "Narrower" : "Wider"}
        >
          {wide ? <Minimize2 className="size-4" /> : <Maximize2 className="size-4" />}
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          nativeButton={false}
          render={<a href={downloadUrl} aria-label="Download" title="Download" />}
        >
          <Download className="size-4" />
        </Button>
        <Button variant="ghost" size="icon" className="size-8" onClick={close} aria-label="Close files">
          <X className="size-4" />
        </Button>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        {target.source === "chat" && files.isPending ? (
          <RowsSkeleton />
        ) : target.source === "chat" && !chatFile ? (
          <PaneMessage icon={<TriangleAlert className="size-5" />} title="This file isn’t in the chat any more" />
        ) : (
          <PreviewBody
            // A new file is a new preview: nothing of the last one's state may
            // carry over, least of all its object URL.
            key={url}
            name={name}
            size={size}
            url={url}
            downloadUrl={downloadUrl}
            mediaType={chatFile?.mediaType}
            cache={target.source === "chat"}
          />
        )}
      </div>
    </>
  );
}

function PreviewBody({
  name,
  size,
  url,
  downloadUrl,
  mediaType,
  cache,
}: {
  name: string;
  size: number;
  url: string;
  downloadUrl: string;
  mediaType?: string;
  /** A chat's files never change; the working folder's can at any moment. */
  cache: boolean;
}) {
  const kind = fileKind(name, mediaType);
  const needsBytes = kind === "pdf" || kind === "text" || kind === "csv" || kind === "markdown";
  const bytes = useQuery({
    queryKey: ["file-bytes", url],
    queryFn: ({ signal }) => api.blob(url, signal),
    enabled: needsBytes,
    staleTime: cache ? Infinity : 0,
    gcTime: cache ? 5 * 60_000 : 0,
    retry: false,
  });

  if (kind === "image")
    return (
      <div className="grid min-h-full place-items-center bg-muted p-4">
        {/* eslint-disable-next-line @next/next/no-img-element -- a preview of our own API's bytes */}
        <img src={url} alt={name} className="max-h-full max-w-full rounded-md shadow-sm" />
      </div>
    );
  if (kind === "audio") return <div className="p-6"><audio controls src={url} className="w-full" /></div>;
  if (kind === "video") return <video controls src={url} className="block max-h-full w-full bg-muted" />;
  if (kind === "other")
    return (
      <PaneMessage icon={<FileKindIcon kind="other" className="size-6" />} title={`No preview for ${name.includes(".") ? `.${name.split(".").pop()}` : "this kind of"} files`}>
        Download it to open it in the app it belongs to.
        <Button variant="outline" size="sm" className="mt-3" nativeButton={false} render={<a href={downloadUrl} />}>
          <Download className="size-3.5" />
          Download
        </Button>
      </PaneMessage>
    );

  if (bytes.isPending) return <RowsSkeleton compact />;
  if (bytes.isError)
    return (
      <PaneMessage icon={<TriangleAlert className="size-5" />} title="Couldn’t show this file">
        {bytes.error.message}
        <Button variant="outline" size="sm" className="mt-3" nativeButton={false} render={<a href={downloadUrl} />}>
          <Download className="size-3.5" />
          Download
        </Button>
      </PaneMessage>
    );
  if (kind === "pdf") return <PdfView blob={bytes.data} name={name} />;
  return <TextView blob={bytes.data} kind={kind} size={size} />;
}

/** The browser's own PDF viewer, given the bytes as a blob: a link straight to
 *  the API would be downloaded rather than shown, since the API serves files to
 *  be kept, not run. */
function PdfView({ blob, name }: { blob: Blob; name: string }) {
  const frame = useRef<HTMLIFrameElement>(null);
  // The address is made and let go with the frame that uses it, so a PDF's
  // bytes are not kept in memory after the preview closes.
  useEffect(() => {
    const u = URL.createObjectURL(new Blob([blob], { type: "application/pdf" }));
    if (frame.current) frame.current.src = u;
    return () => URL.revokeObjectURL(u);
  }, [blob]);
  return <iframe ref={frame} title={name} className="block h-full min-h-[480px] w-full border-0 bg-muted" />;
}

function TextView({ blob, kind, size }: { blob: Blob; kind: "text" | "csv" | "markdown"; size: number }) {
  // Decoding is local work on bytes already fetched, not server state.
  const [decoded, setDecoded] = useState<{ blob: Blob; text: string } | null>(null);
  useEffect(() => {
    let live = true;
    void blob
      .slice(0, TEXT_LIMIT)
      .text()
      .then((t) => live && setDecoded({ blob, text: t }));
    return () => {
      live = false;
    };
  }, [blob]);
  const text = { data: decoded?.blob === blob ? decoded.text : undefined };
  const cut = blob.size > TEXT_LIMIT;
  const note = cut && (
    <p className="px-4 pb-4 text-xs text-muted-foreground">
      Showing the first {formatBytes(TEXT_LIMIT)} of {formatBytes(size || blob.size)}. Download it to see the rest.
    </p>
  );
  if (!text.data) return <RowsSkeleton compact />;
  if (kind === "markdown")
    return (
      <div className="px-5 py-4">
        <Markdown text={text.data} />
        {note}
      </div>
    );
  if (kind === "csv") return <><CsvTable text={text.data} />{note}</>;
  return (
    <>
      <pre className="m-0 py-3 font-mono text-xs leading-[19px]">
        {text.data.replace(/\r?\n$/, "").split("\n").map((line, i) => (
          <div key={i} className="flex">
            <span className="w-11 shrink-0 pr-3 text-right text-muted-foreground/60 select-none">{i + 1}</span>
            <span className="pr-4 whitespace-pre">{line}</span>
          </div>
        ))}
      </pre>
      {note}
    </>
  );
}

/** Enough of a CSV to read: quoted fields, commas and doubled quotes inside
 *  them, the first 500 rows. */
function CsvTable({ text }: { text: string }) {
  const rows = useMemo(() => parseCsv(text, 501), [text]);
  if (rows.length === 0) return <PaneMessage icon={<FileKindIcon kind="csv" className="size-5" />} title="This file is empty" />;
  const [head, ...body] = rows;
  return (
    <div className="p-3">
      <table className="border-collapse text-xs">
        <thead>
          <tr>
            {head.map((c, i) => (
              <th key={i} className="border bg-muted px-2.5 py-1 text-left font-medium whitespace-nowrap">{c}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {body.slice(0, 500).map((r, i) => (
            <tr key={i}>
              {r.map((c, j) => (
                <td key={j} className={cn("border px-2.5 py-1 whitespace-nowrap", /^-?[\d.,]+$/.test(c) && "text-right font-mono")}>{c}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <p className="mt-2 text-xs text-muted-foreground">
        {body.length >= 500 ? "The first 500 rows" : `${body.length} rows`} · {head.length} columns
      </p>
    </div>
  );
}

export function parseCsv(text: string, maxRows: number): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let field = "";
  let quoted = false;
  for (let i = 0; i < text.length && rows.length < maxRows; i++) {
    const ch = text[i];
    if (quoted) {
      if (ch === '"' && text[i + 1] === '"') {
        field += '"';
        i++;
      } else if (ch === '"') quoted = false;
      else field += ch;
    } else if (ch === '"') quoted = true;
    else if (ch === ",") {
      row.push(field);
      field = "";
    } else if (ch === "\n" || ch === "\r") {
      if (ch === "\r" && text[i + 1] === "\n") i++;
      row.push(field);
      rows.push(row);
      row = [];
      field = "";
    } else field += ch;
  }
  if ((field || row.length) && rows.length < maxRows) {
    row.push(field);
    rows.push(row);
  }
  return rows;
}

// ---------------------------------------------------------------------------

function PaneMessage({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center px-7 py-12 text-center text-sm text-muted-foreground">
      {icon}
      <p className="mt-2.5 mb-1 font-medium text-foreground">{title}</p>
      {children}
    </div>
  );
}

function RowsSkeleton({ compact }: { compact?: boolean }) {
  return (
    <div className="space-y-2 p-3" aria-busy="true" aria-label="Loading">
      {[0, 1, 2, 3].map((i) =>
        compact ? (
          <Skeleton key={i} className="h-4" style={{ width: `${70 - i * 9}%` }} />
        ) : (
          <div key={i} className="flex items-center gap-2.5">
            <Skeleton className="size-9 shrink-0" />
            <div className="flex-1 space-y-1.5">
              <Skeleton className="h-3 w-3/5" />
              <Skeleton className="h-2.5 w-2/5" />
            </div>
          </div>
        ),
      )}
    </div>
  );
}
