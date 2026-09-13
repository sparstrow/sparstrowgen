import { ResetPassword } from "@/components/auth/email-link";

/* Opened from the password-reset email. See verify/page.tsx for why a missing
   token is passed on rather than handled here. */
export default async function ResetPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string | string[] }>;
}) {
  const { token } = await searchParams;
  return <ResetPassword token={typeof token === "string" ? token : ""} />;
}
