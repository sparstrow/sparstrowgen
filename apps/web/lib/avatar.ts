"use client";

/* Turning whatever a person picked into something worth storing.
 *
 * Phone cameras produce several megabytes; the server caps an avatar at 512KB
 * and it is shown at 28px in the rail. Sending the original would be slow on a
 * phone connection, would often be refused outright, and would store a photo a
 * hundred times larger than anything that is ever displayed. So the browser
 * squares it and shrinks it first, and the server's limit is the backstop for a
 * client that did not (store/profile.go).
 *
 * WebP where the browser can encode it, PNG otherwise. Both are on the server's
 * accepted list, and neither is decided by the file that came in: a PNG that is
 * really a photo becomes a small WebP, and the stored type is whatever was
 * actually encoded rather than whatever was claimed. */

/** The stored size. Four times the largest place it is shown (64px), so it
 *  stays sharp on a high-density screen without storing a photograph. */
const SIDE = 256;

export const ACCEPTED_IMAGE_TYPES = "image/png,image/jpeg,image/webp";

export class NotAnImage extends Error {
  constructor() {
    super("That file is not an image we can read. Try a PNG, JPEG or WebP.");
  }
}

/** Centre-cropped to a square and scaled to SIDE. Centre rather than top,
 *  because a portrait photo's subject is in the middle far more often than not,
 *  and there is no crop tool here to correct a bad guess. */
export async function squareImage(file: File): Promise<Blob> {
  const bitmap = await readBitmap(file);
  try {
    const side = Math.min(bitmap.width, bitmap.height);
    const sx = (bitmap.width - side) / 2;
    const sy = (bitmap.height - side) / 2;

    const canvas = document.createElement("canvas");
    // Never upscale: a 64px picture stored at 256 is four times the bytes for
    // exactly the same amount of detail.
    canvas.width = canvas.height = Math.min(SIDE, side);
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new NotAnImage();
    ctx.drawImage(bitmap, sx, sy, side, side, 0, 0, canvas.width, canvas.height);

    return await encode(canvas);
  } finally {
    // Frees the decoded pixels rather than waiting for a collection that may
    // not come until several photos have been decoded.
    bitmap.close?.();
  }
}

async function readBitmap(file: File): Promise<ImageBitmap> {
  // createImageBitmap decodes off the main thread and is what every current
  // browser has; the <img> path is the fallback, not the plan.
  if (typeof createImageBitmap === "function") {
    try {
      return await createImageBitmap(file);
    } catch {
      throw new NotAnImage();
    }
  }
  const url = URL.createObjectURL(file);
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image();
      el.onload = () => resolve(el);
      el.onerror = () => reject(new NotAnImage());
      el.src = url;
    });
    return (await createImageBitmap(img)) as ImageBitmap;
  } finally {
    URL.revokeObjectURL(url);
  }
}

function encode(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const done = (blob: Blob | null) => {
      if (!blob) return reject(new NotAnImage());
      resolve(blob);
    };
    // toBlob falls back to PNG on its own when a type is not supported, and the
    // resulting blob carries the type that was ACTUALLY encoded — which is what
    // gets sent as Content-Type, so the two can never disagree.
    canvas.toBlob(done, "image/webp", 0.85);
  });
}

/** One or two letters for an account with no picture. From the name when there
 *  is one, otherwise the email — never empty, because a blank circle reads as
 *  something failing to load rather than as nobody having chosen a picture. */
export function initials(displayName: string | undefined, email: string | undefined): string {
  const name = (displayName ?? "").trim();
  if (name) {
    const parts = name.split(/\s+/).filter(Boolean);
    if (parts.length > 1) return (first(parts[0]) + first(parts[parts.length - 1])).toUpperCase();
    return first(parts[0], 2).toUpperCase();
  }
  const local = (email ?? "").split("@")[0] ?? "";
  return (first(local, 2) || "?").toUpperCase();
}

/** The first n CHARACTERS, not bytes or code units — so an emoji or an accented
 *  letter counts as one and is never cut in half. */
function first(s: string, n = 1): string {
  return [...(s ?? "")].slice(0, n).join("");
}
