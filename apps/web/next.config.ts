import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* Standalone output: a self-contained server plus only the modules Next
     actually traced, rather than the whole dependency tree. It is what makes
     Dockerfile.web's runtime stage able to skip node_modules entirely. */
  output: "standalone",

  /* The trace has to start at the WORKSPACE root, not at apps/web.
     Next infers a root by walking up for a lockfile, and in a pnpm workspace it
     can pick the wrong one and leave out symlinked dependencies — which fails
     at runtime, inside the container, as a module that cannot be found, rather
     than at build time where it would be obvious. Saying it explicitly costs
     nothing and removes the guess. */
  outputFileTracingRoot: __dirname + "/../..",
};

export default nextConfig;
