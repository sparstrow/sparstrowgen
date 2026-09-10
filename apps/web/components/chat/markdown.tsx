"use client";

import { memo, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import { Check, Copy } from "lucide-react";

/* Agent answers are markdown. Rendering them as plain text flattened every
   fenced block, heading and list into one paragraph — which on a chat for
   CODING agents threw away most of the value of the reply (docs/Bugs.md B-2).

   Memoised on the text. That does not save the message currently streaming —
   its text changes on every delta, and claude sends one about every eight
   characters — but it stops every OTHER message in the transcript re-parsing
   alongside it, which is where the cost would actually have been. */

/* ── The fence's own words ────────────────────────────────────────────────────
   A fence carries more than code: ```go says the language, and anything after
   it (```go title="main.go") is the meta string. Both were being dropped on the
   floor (docs/Bugs.md B-4).

   The language has to be captured BEFORE rehype-highlight runs. remark turns
   ```go into class="language-go", and when `detect` is on the highlighter adds
   a class of exactly that shape to untagged fences after GUESSING at them — so
   afterwards the agent's word and the machine's guess are indistinguishable.
   Labelling a guess as if the agent had written it is the kind of quiet lie
   that makes a header worse than no header, so the declared value is stashed
   while it is still known to be declared.

   Minimal local node types rather than @types/mdast: three fields are used, and
   importing a transitive dependency's types directly is a phantom import. */

type MdastNode = {
  type: string;
  lang?: string | null;
  data?: { hProperties?: Record<string, string> };
  children?: MdastNode[];
};

function walk(node: MdastNode, fn: (n: MdastNode) => void) {
  fn(node);
  for (const child of node.children ?? []) walk(child, fn);
}

function remarkDeclaredLanguage() {
  return (tree: MdastNode) => {
    walk(tree, (node) => {
      if (node.type !== "code" || !node.lang) return;
      const data = (node.data ??= {});
      data.hProperties = { ...data.hProperties, "data-lang": node.lang };
    });
  };
}

/** The `<code>` element inside a `<pre>`, as react-markdown hands the node over.
 *  `data.meta` is set by mdast-util-to-hast itself; `data-lang` is ours. */
type CodeNode = {
  type?: string;
  properties?: Record<string, unknown>;
  data?: { meta?: string | null };
};

function fenceOf(node: unknown): { lang?: string; info?: string } {
  const code = (node as { children?: CodeNode[] } | undefined)?.children?.[0];
  if (!code || code.type !== "element") return {};
  const lang = code.properties?.["data-lang"];
  const info = code.data?.meta;
  return {
    lang: typeof lang === "string" ? lang : undefined,
    info: typeof info === "string" && info.trim() ? info.trim() : undefined,
  };
}

/** Display names for the tags agents actually write. Anything not listed shows
 *  the tag verbatim — a language we have never seen is still worth naming, and
 *  a wrong expansion would be worse than the tag itself. */
const LANGUAGE_NAMES: Record<string, string> = {
  bash: "Bash",
  c: "C",
  cpp: "C++",
  cs: "C#",
  css: "CSS",
  diff: "Diff",
  dockerfile: "Dockerfile",
  go: "Go",
  html: "HTML",
  java: "Java",
  js: "JavaScript",
  javascript: "JavaScript",
  json: "JSON",
  jsx: "JSX",
  kotlin: "Kotlin",
  makefile: "Makefile",
  md: "Markdown",
  markdown: "Markdown",
  php: "PHP",
  ps1: "PowerShell",
  powershell: "PowerShell",
  proto: "Protobuf",
  py: "Python",
  python: "Python",
  rb: "Ruby",
  ruby: "Ruby",
  rs: "Rust",
  rust: "Rust",
  sh: "Shell",
  shell: "Shell",
  sql: "SQL",
  swift: "Swift",
  toml: "TOML",
  ts: "TypeScript",
  typescript: "TypeScript",
  tsx: "TSX",
  xml: "XML",
  yaml: "YAML",
  yml: "YAML",
};

function CodeBlock({
  lang,
  info,
  children,
}: {
  lang?: string;
  info?: string;
  children: React.ReactNode;
}) {
  const [copied, setCopied] = useState(false);

  async function copy(e: React.MouseEvent<HTMLButtonElement>) {
    const code =
      e.currentTarget
        .closest("[data-slot='code-block']")
        ?.querySelector("code")?.textContent ?? "";
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard access can be refused. Saying nothing is better than an
      // error toast for something the user can still select by hand.
    }
  }

  return (
    <div
      data-slot="code-block"
      className="my-3 overflow-hidden rounded-lg border"
    >
      {/* The bar exists for the copy button whether or not a language was
          declared, so an untagged fence simply leaves the left side empty
          rather than being labelled with a guess. */}
      <div className="flex items-center gap-2 border-b bg-muted/40 px-3 py-1.5">
        {lang && (
          <span className="font-mono text-xs text-foreground">
            {LANGUAGE_NAMES[lang.toLowerCase()] ?? lang}
          </span>
        )}
        {info && (
          <span className="truncate font-mono text-xs text-muted-foreground">
            {info}
          </span>
        )}
        <button
          type="button"
          onClick={copy}
          className="ml-auto flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
        >
          {copied ? (
            <>
              <Check className="size-3.5" aria-hidden />
              Copied
            </>
          ) : (
            <>
              <Copy className="size-3.5" aria-hidden />
              Copy
            </>
          )}
        </button>
      </div>
      {/* Wide code scrolls inside its own box; the transcript column never
          scrolls sideways. */}
      <pre className="overflow-x-auto bg-background/60 p-3.5 text-[13px] leading-relaxed">
        {children}
      </pre>
    </div>
  );
}

export const Markdown = memo(function Markdown({ text }: { text: string }) {
  return (
    <div className="text-[15px] leading-relaxed">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkDeclaredLanguage]}
        rehypePlugins={[[rehypeHighlight, { detect: true, ignoreMissing: true }]]}
        components={{
          // `pre` carries the header and the copy button; `code` inside it is
          // left alone so the highlighter's spans survive.
          pre: ({ node, children }) => {
            const { lang, info } = fenceOf(node);
            return (
              <CodeBlock lang={lang} info={info}>
                {children}
              </CodeBlock>
            );
          },
          code: ({ className, children, ...props }) => {
            const fenced = /language-/.test(className ?? "");
            if (fenced) {
              return (
                <code className={`font-mono ${className ?? ""}`} {...props}>
                  {children}
                </code>
              );
            }
            return (
              <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[13px]">
                {children}
              </code>
            );
          },
          p: ({ children }) => <p className="my-3 first:mt-0 last:mb-0">{children}</p>,
          h1: ({ children }) => (
            <h1 className="mt-5 mb-2 text-base font-semibold first:mt-0">{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className="mt-5 mb-2 text-[15px] font-semibold first:mt-0">{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="mt-4 mb-1.5 text-sm font-semibold first:mt-0">{children}</h3>
          ),
          ul: ({ children }) => (
            <ul className="my-3 list-disc space-y-1 pl-5">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="my-3 list-decimal space-y-1 pl-5">{children}</ol>
          ),
          li: ({ children }) => <li className="pl-0.5">{children}</li>,
          a: ({ children, href }) => (
            <a
              href={href}
              target="_blank"
              rel="noreferrer noopener"
              className="underline underline-offset-2 hover:text-foreground"
            >
              {children}
            </a>
          ),
          blockquote: ({ children }) => (
            <blockquote className="my-3 border-l-2 pl-3.5 text-muted-foreground">
              {children}
            </blockquote>
          ),
          hr: () => <hr className="my-4" />,
          // A table wider than the column scrolls itself rather than pushing
          // the page sideways.
          table: ({ children }) => (
            <div className="my-3 overflow-x-auto">
              <table className="w-full border-collapse text-sm">{children}</table>
            </div>
          ),
          th: ({ children }) => (
            <th className="border px-2.5 py-1.5 text-left font-medium">{children}</th>
          ),
          td: ({ children }) => <td className="border px-2.5 py-1.5">{children}</td>,
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  );
});
