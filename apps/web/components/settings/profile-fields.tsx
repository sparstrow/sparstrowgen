"use client";

import { useRef, useState } from "react";
import { Trash2, Upload } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { ACCEPTED_IMAGE_TYPES, initials, NotAnImage, squareImage } from "@/lib/avatar";
import { useProfile, useRemoveAvatar, useSaveAvatar, useSaveProfile, useSession } from "@/lib/queries";
import { Avatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

/* The profile editor itself, so Settings → Account and the first-run setup step
   are the same fields rather than two forms that drift (the same reasoning as
   D-046 for the step list).

   The picture saves on choosing it and the text saves on Save. That is not an
   inconsistency: picking a file is already a deliberate confirmed action with
   its own dialog, and making someone press Save afterwards to keep a picture
   they can see in front of them invites closing the page with it unsaved. */

export const NAME_LIMIT = 80;
export const BIO_LIMIT = 400;

export function ProfileFields({ onSaved }: { onSaved?: () => void }) {
  const session = useSession();
  const profile = useProfile();
  const save = useSaveProfile();
  const setAvatar = useSaveAvatar();
  const removeAvatar = useRemoveAvatar();
  const fileRef = useRef<HTMLInputElement>(null);

  const [name, setName] = useState("");
  const [bio, setBio] = useState("");
  const saved = profile.data;

  /* The server is the source of truth and these inputs start from it, but a
     refetch must not throw away half-typed text.

     Keyed on the saved VALUES, not on the query result's identity: the cache
     hands back a new object on every refetch, so keying on the object would
     reset the form under the cursor each time. Adjusting state during render
     rather than in an effect is React's own answer to "reset state when a value
     changes", and it avoids a render with the stale text still on screen. */
  const savedKey = JSON.stringify([saved?.displayName ?? "", saved?.bio ?? ""]);
  const [seen, setSeen] = useState<string | null>(null);
  if (saved && savedKey !== seen) {
    setSeen(savedKey);
    setName(saved.displayName);
    setBio(saved.bio);
  }

  const src = api.avatarUrl(saved?.avatarUpdatedAt);
  const letters = initials(saved?.displayName, session.data?.email);
  const dirty = name !== (saved?.displayName ?? "") || bio !== (saved?.bio ?? "");
  const tooLong = name.length > NAME_LIMIT || bio.length > BIO_LIMIT;
  const busy = setAvatar.isPending || removeAvatar.isPending;

  async function choose(file: File | undefined) {
    if (!file) return;
    try {
      const square = await squareImage(file);
      await setAvatar.mutateAsync(square);
      toast.success("Picture updated");
    } catch (e) {
      toast.error(
        e instanceof NotAnImage ? e.message : "That picture could not be saved",
        { description: e instanceof NotAnImage ? undefined : (e as Error).message },
      );
    } finally {
      // Cleared so that choosing the SAME file again still fires a change.
      if (fileRef.current) fileRef.current.value = "";
    }
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    save.mutate(
      { displayName: name, bio },
      {
        onSuccess: () => {
          toast.success("Profile saved");
          onSaved?.();
        },
        onError: (err) =>
          toast.error("Your profile could not be saved", { description: err.message }),
      },
    );
  }

  return (
    <form onSubmit={submit} className="space-y-6">
      <div className="flex items-center gap-4">
        <Avatar src={src} initials={letters} className="size-16 text-lg" alt="Your picture" />
        <div className="min-w-0 space-y-2">
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => fileRef.current?.click()}
            >
              <Upload />
              {src ? "Change picture" : "Add a picture"}
            </Button>
            {src && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={busy}
                onClick={() =>
                  removeAvatar.mutate(undefined, {
                    onSuccess: () => toast.success("Picture removed"),
                    onError: (err) =>
                      toast.error("That picture could not be removed", {
                        description: err.message,
                      }),
                  })
                }
              >
                <Trash2 />
                Remove
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            PNG, JPEG or WebP. It is squared and shrunk in your browser before it is sent.
          </p>
        </div>
        <input
          ref={fileRef}
          type="file"
          accept={ACCEPTED_IMAGE_TYPES}
          className="hidden"
          onChange={(e) => void choose(e.target.files?.[0])}
        />
      </div>

      <div className="space-y-2">
        <label htmlFor="display-name" className="text-sm font-medium">
          Name
        </label>
        <Input
          id="display-name"
          value={name}
          maxLength={NAME_LIMIT}
          onChange={(e) => setName(e.target.value)}
          placeholder={session.data?.email ?? "What you would like to be called"}
          autoComplete="name"
        />
        <p className="text-xs text-muted-foreground">
          Shown wherever the app refers to you. Leave it empty to use your email address.
        </p>
      </div>

      <div className="space-y-2">
        <label htmlFor="bio" className="text-sm font-medium">
          About you
        </label>
        <Textarea
          id="bio"
          value={bio}
          maxLength={BIO_LIMIT}
          rows={3}
          onChange={(e) => setBio(e.target.value)}
          placeholder="A line about what you work on. Optional."
        />
        <p className="text-xs text-muted-foreground">
          {bio.length} of {BIO_LIMIT} characters
        </p>
      </div>

      <div className="flex items-center gap-3">
        <Button type="submit" disabled={!dirty || tooLong || save.isPending}>
          {save.isPending ? "Saving…" : "Save"}
        </Button>
        {dirty && !save.isPending && (
          <span className="text-xs text-muted-foreground">Not saved yet</span>
        )}
      </div>
    </form>
  );
}
