"use client";

import { useState } from "react";
import { cn } from "cn";

/* A person, in a circle. The picture when there is one, their initials when
   there is not — never an empty circle, which reads as something that failed to
   load rather than as a choice nobody made.

   No shadcn Avatar in this app's registry, and this needs less than one: a
   single image, one fallback, no async primitive. */

export function Avatar({
  src,
  initials,
  className,
  alt = "",
}: {
  src: string | null;
  initials: string;
  className?: string;
  /** Empty by default: beside a name, the picture adds nothing a screen reader
   *  needs, and "photo of X" next to "X" is noise. Set it where the picture
   *  stands alone. */
  alt?: string;
}) {
  // A stored picture can fail to load — offline, or removed in another tab
  // between the profile being read and the image being fetched. Falling back to
  // the initials keeps a person in the circle either way.
  //
  // Which src failed, rather than a boolean plus an effect to clear it: a new
  // picture is a new URL, so it is simply not the failed one and shows without
  // anything having to reset.
  const [failed, setFailed] = useState<string | null>(null);
  const show = src && failed !== src;

  return (
    <span
      className={cn(
        "inline-flex size-7 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-[11px] font-medium text-muted-foreground select-none",
        className,
      )}
    >
      {show ? (
        /* next/image wants a configured loader and a known host; this is one
           small same-origin image behind a session cookie, already sized to
           256px by the browser that uploaded it. */
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={src}
          alt={alt}
          className="size-full object-cover"
          onError={() => setFailed(src)}
        />
      ) : (
        <span aria-hidden={alt === "" ? true : undefined}>{initials}</span>
      )}
    </span>
  );
}
