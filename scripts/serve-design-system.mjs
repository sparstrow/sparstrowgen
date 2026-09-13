// Serves design-system/ over http so prototypes resolve their relative
// stylesheet and seed-data paths. A file opened as a data: or sandboxed URL
// cannot, and renders unstyled with no data. Zero dependencies.
//
//   node scripts/serve-design-system.mjs [port]   (default 4173)
import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "design-system");
const port = Number(process.argv[2] ?? process.env.PORT ?? 4173);
const types = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".md": "text/markdown; charset=utf-8",
};

http
  .createServer((req, res) => {
    const url = decodeURIComponent(new URL(req.url, "http://x").pathname);
    const file = path.resolve(root, "." + (url.endsWith("/") ? url + "index.html" : url));
    if (!file.startsWith(root)) {
      res.writeHead(403).end("outside design-system");
      return;
    }
    fs.readFile(file, (err, body) => {
      if (err) {
        res.writeHead(404, { "content-type": "text/plain" }).end("not found: " + url);
        return;
      }
      res.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream", "cache-control": "no-store" });
      res.end(body);
    });
  })
  .listen(port, "127.0.0.1", () => console.log(`design-system on http://localhost:${port}`));
