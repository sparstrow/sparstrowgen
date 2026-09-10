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

function CodeBlock({ children }: { children: React.ReactNode }) {
  const [copied, setCopied] = useState(false);

  async function copy(e: React.MouseEvent<HTMLButtonElement>) {
    const code = e.currentTarget.parentElement?.querySelector("code")?.textContent ?? "";
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
    <div className="group/code relative my-3">
      <button
        type="button"
        onClick={copy}
        aria-label={copied ? "Copied" : "Copy code"}
        className="absolute right-2 top-2 rounded-md border bg-background/80 p-1.5 opacity-0 backdrop-blur transition-opacity group-hover/code:opacity-100 focus-visible:opacity-100"
      >
        {copied ? (
          <Check className="size-3.5 text-foreground" />
        ) : (
          <Copy className="size-3.5 text-muted-foreground" />
        )}
      </button>
      {/* Wide code scrolls inside its own box; the transcript column never
          scrolls sideways. */}
      <pre className="overflow-x-auto rounded-lg border bg-background/60 p-3.5 text-[13px] leading-relaxed">
        {children}
      </pre>
    </div>
  );
}

export const Markdown = memo(function Markdown({ text }: { text: string }) {
  return (
    <div className="text-[15px] leading-relaxed">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[[rehypeHighlight, { detect: true, ignoreMissing: true }]]}
        components={{
          // `pre` carries the copy button; `code` inside it is left alone so
          // the highlighter's spans survive.
          pre: ({ children }) => <CodeBlock>{children}</CodeBlock>,
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
