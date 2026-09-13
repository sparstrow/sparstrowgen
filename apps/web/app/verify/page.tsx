import { ConfirmEmail } from "@/components/auth/email-link";

/* Opened from the confirmation email. A missing or repeated token param is
   passed on as an empty string, which the server reports as an expired link
   rather than this page inventing a separate "broken link" state. */
export default async function VerifyPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string | string[] }>;
}) {
  const { token } = await searchParams;
  return <ConfirmEmail token={typeof token === "string" ? token : ""} />;
}
