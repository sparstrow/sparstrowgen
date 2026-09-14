"use client";
import { useParams } from "next/navigation";
import { MachineProfile } from "@/components/machines/machines-surface";
export default function MachinePage(){const {id}=useParams<{id:string}>();return <MachineProfile id={id}/>}
