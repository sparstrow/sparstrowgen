import Link from "next/link";
import { Download, MonitorSmartphone } from "lucide-react";
import { Button } from "@/components/ui/button";

const installer = process.env.NEXT_PUBLIC_WINDOWS_INSTALLER_URL;

export default function InstallPage() {
  return <main className="mx-auto flex min-h-full w-full max-w-xl flex-col justify-center px-6 py-12"><MonitorSmartphone className="size-7" aria-hidden/><h1 className="mt-5 text-lg font-medium tracking-tight">Install sparstrowgen on this computer</h1><p className="mt-2 text-sm text-muted-foreground">Install the Windows component here, then return to Machines and choose Add computer again. It only connects out to sparstrowgen and uses the coding-agent tools already installed for your Windows account.</p>{installer ? <Button className="mt-6 w-fit" render={<a href={installer}/> }><Download/>Download Windows installer</Button> : <p className="mt-6 border px-4 py-3 text-sm text-muted-foreground">The signed Windows installer has not been published to this environment yet. It is safe to retry pairing once the installer is available.</p>}<Button className="mt-5 w-fit" variant="outline" render={<Link href="/machines"/>}>Back to Machines</Button></main>;
}
