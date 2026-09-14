"use client";

import Link from "next/link";
import { MessageSquareText, MonitorSmartphone, Settings } from "lucide-react";
import {
  Sidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupContent, SidebarHeader,
  SidebarMenu, SidebarMenuButton, SidebarMenuItem,
} from "@/components/ui/sidebar";
import { AccountMenu } from "@/components/auth/account-menu";

export function ProductSidebar({ current }: { current: "chat" | "machines" | "settings" }) {
  return <Sidebar collapsible="icon">
    <SidebarHeader className="px-3 py-4 text-sm font-semibold tracking-tight">sparstrowgen</SidebarHeader>
    <SidebarContent><SidebarGroup><SidebarGroupContent><SidebarMenu>
      <SidebarMenuItem><SidebarMenuButton render={<Link href="/" />} isActive={current === "chat"} tooltip="Chat"><MessageSquareText /> <span>Chat</span></SidebarMenuButton></SidebarMenuItem>
      <SidebarMenuItem><SidebarMenuButton render={<Link href="/machines" />} isActive={current === "machines"} tooltip="Machines"><MonitorSmartphone /> <span>Machines</span></SidebarMenuButton></SidebarMenuItem>
      <SidebarMenuItem><SidebarMenuButton render={<Link href="/settings" />} isActive={current === "settings"} tooltip="Settings"><Settings /> <span>Settings</span></SidebarMenuButton></SidebarMenuItem>
    </SidebarMenu></SidebarGroupContent></SidebarGroup></SidebarContent>
    <SidebarFooter className="items-center group-data-[state=expanded]:items-stretch">
      <AccountMenu />
    </SidebarFooter>
  </Sidebar>;
}
