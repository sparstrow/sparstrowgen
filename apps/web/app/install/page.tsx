import Link from "next/link";
import { Download, MonitorSmartphone } from "lucide-react";
import { Button } from "@/components/ui/button";

const installer =
  process.env.NEXT_PUBLIC_WINDOWS_INSTALLER_URL ??
  "https://github.com/sparstrow/sparstrowgen/releases/latest/download/sparstrowgen-setup.exe";

export default function InstallPage() {
  return (
    <main className="mx-auto flex min-h-full w-full max-w-xl flex-col justify-center px-6 py-12">
      <MonitorSmartphone className="size-7" aria-hidden />
      <h1 className="mt-5 text-lg font-medium tracking-tight">Install sparstrowgen on this computer</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Use the Windows computer where your coding agents are installed. sparstrowgen sets itself up for your Windows
        account, starts when you sign in, and only connects out to sparstrowgen.
      </p>
      <Button className="mt-6 w-fit" render={<a href={installer} />}>
        <Download />
        Download for Windows
      </Button>
      <ol className="mt-6 list-decimal space-y-2 pl-5 text-sm text-muted-foreground">
        <li>Open sparstrowgen-setup.exe from your downloads.</li>
        <li>
          If Windows says it protected your PC, choose More info, then Run anyway. The installer is not signed yet.
        </li>
        <li>When it says sparstrowgen is installed, come back to Machines and choose Add computer.</li>
      </ol>
      <Button className="mt-6 w-fit" variant="outline" nativeButton={false} render={<Link href="/machines" />}>
        Back to Machines
      </Button>
    </main>
  );
}
