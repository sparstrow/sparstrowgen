import * as React from "react"
import {
  CircleCheck,
  CircleMinus,
  Clock,
  Info,
  Loader2,
  OctagonX,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react"
import { cn } from "cn"
import { Badge, type badgeVariants } from "@/components/ui/badge"
import type { VariantProps } from "class-variance-authority"

/* The one place a status gets its icon.

   A status is three things at once: a colour, a shape, and a word. Colour alone
   fails for anyone who cannot tell green from red, and shape alone fails for
   anyone who has not learned the shapes, so every status carries all three. The
   silhouettes are deliberately different (circle with a tick, circle with an i,
   triangle, octagon, spinner, clock, circle with a dash), so the tone can be told
   apart with the colour taken away.

   There is no `icon` prop on purpose. A screen that wants a different glyph for a
   status is describing a status this list does not have yet, and the fix is to
   add it here so every other screen gets it too. */

export type StatusTone =
  | "success"
  | "info"
  | "warning"
  | "danger"
  | "progress"
  | "pending"
  | "neutral"

type BadgeVariant = NonNullable<VariantProps<typeof badgeVariants>["variant"]>

const TONES: Record<
  StatusTone,
  {
    icon: LucideIcon
    /** The icon's own colour, on a plain surface. */
    iconClass: string
    /** The word's colour: the -text token, which holds 4.5:1 where the icon colour does not. */
    textClass: string
    /** Only the progress spinner turns. `motion-safe` so reduced motion stops it. */
    spinClass?: string
    badge: BadgeVariant
  }
> = {
  // Done, working, available, online.
  success: {
    icon: CircleCheck,
    iconClass: "text-success",
    textClass: "text-success-text",
    badge: "success",
  },
  // Newly available, or worth knowing. Nothing to do about it yet.
  info: {
    icon: Info,
    iconClass: "text-info",
    textClass: "text-info-text",
    badge: "info",
  },
  // Needs a person to act, and nothing is broken: not installed, needs an update.
  warning: {
    icon: TriangleAlert,
    iconClass: "text-warning",
    textClass: "text-warning-text",
    badge: "warning",
  },
  // A fault, or something that failed.
  danger: {
    icon: OctagonX,
    iconClass: "text-destructive",
    textClass: "text-destructive-text",
    badge: "destructive",
  },
  // Happening right now.
  progress: {
    icon: Loader2,
    iconClass: "text-info",
    textClass: "text-info-text",
    spinClass: "motion-safe:animate-spin",
    badge: "info",
  },
  // Waiting for something to finish. Nothing is wrong, and nothing to do.
  pending: {
    icon: Clock,
    iconClass: "text-warning",
    textClass: "text-warning-text",
    badge: "warning",
  },
  // Absent, off, unknown. Not a fault, so it stays muted.
  neutral: {
    icon: CircleMinus,
    iconClass: "text-muted-foreground",
    textClass: "text-muted-foreground",
    badge: "secondary",
  },
}

function StatusIcon({
  tone,
  size = "sm",
  className,
}: {
  tone: StatusTone
  size?: "sm" | "md"
  className?: string
}) {
  const t = TONES[tone]
  const Icon = t.icon
  return (
    <Icon
      aria-hidden="true"
      className={cn(
        "shrink-0",
        size === "md" ? "size-4" : "size-3.5",
        t.iconClass,
        t.spinClass,
        className
      )}
    />
  )
}

type StatusProps = Omit<React.ComponentProps<"span">, "children"> & {
  tone: StatusTone
  /** The word. Always shown: a status is never an icon on its own. */
  children: React.ReactNode
  /** `inline` is an icon and a word in the running text. `badge` puts them in a pill. */
  appearance?: "inline" | "badge"
  size?: "sm" | "md"
  /** The word keeps the surrounding text colour and only the icon carries the tone.
   *  For a sentence; a one-word label reads better coloured. */
  quiet?: boolean
}

function Status({
  tone,
  children,
  appearance = "inline",
  size = "sm",
  quiet = false,
  className,
  ...props
}: StatusProps) {
  const t = TONES[tone]

  if (appearance === "badge") {
    const Icon = t.icon
    return (
      <Badge variant={t.badge} className={className} {...props}>
        <Icon aria-hidden="true" className={t.spinClass} />
        {children}
      </Badge>
    )
  }

  return (
    <span
      className={cn("inline-flex items-start gap-1.5", className)}
      {...props}
    >
      {/* One line tall, so the icon sits on the first line of a sentence that wraps. */}
      <span className="flex h-lh shrink-0 items-center">
        <StatusIcon tone={tone} size={size} />
      </span>
      <span className={cn("min-w-0", !quiet && t.textClass)}>{children}</span>
    </span>
  )
}

export { Status, StatusIcon }
