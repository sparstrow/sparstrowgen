import type { ComputerUpdates, UpdateStatus } from "@/lib/chat-types";

// Mock data for the US3 Updates page while its design is confirmed. Deleted when
// the backend lands; a shipped route importing this file is not done.

export const updateScenarios = ["waiting", "current", "available", "updating", "failed", "too-old", "offline", "two-computers", "empty", "loading", "error"] as const;
export type UpdateScenario = (typeof updateScenarios)[number];

export function parseScenario(raw: string | null): UpdateScenario {
  return updateScenarios.find((s) => s === raw) ?? "waiting";
}

const latest = "0.1.3";
const activeTasks: Record<string, number> = { "m-desktop": 2, "m-laptop": 0 };

const desktop = (status: UpdateStatus, extra: Partial<ComputerUpdates> = {}): ComputerUpdates => ({
  machineId: "m-desktop", name: "DESKTOP-GJ8NLB8", online: true, version: "0.1.2", tooOld: false, automatic: true, status, ...extra,
});

const laptop: ComputerUpdates = {
  machineId: "m-laptop", name: "WORK-LAPTOP-02", online: false, version: "0.1.1", tooOld: false, automatic: false, status: { kind: "unchecked" },
};

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export async function fetchComputerUpdates(scenario: UpdateScenario): Promise<ComputerUpdates[]> {
  if (scenario === "loading") return new Promise(() => {});
  await wait(350);
  switch (scenario) {
    case "error": throw new Error("The server could not be reached.");
    case "empty": return [];
    case "current": return [desktop({ kind: "current" }, { version: latest })];
    case "available": return [desktop({ kind: "available", version: latest }, { automatic: false })];
    case "updating": return [desktop({ kind: "updating", version: latest })];
    case "failed": return [desktop({ kind: "failed", message: `v${latest} did not reconnect within 2 minutes, so v0.1.2 was put back and is running.` })];
    case "too-old": return [desktop({ kind: "unchecked" }, { version: "0.1.0", tooOld: true })];
    case "offline": return [desktop({ kind: "unchecked" }, { online: false })];
    case "two-computers": return [desktop({ kind: "waiting", version: latest, activeTasks: 2 }), laptop];
    default: return [desktop({ kind: "waiting", version: latest, activeTasks: 2 })];
  }
}

export async function checkForUpdates(computer: ComputerUpdates): Promise<UpdateStatus> {
  await wait(1200);
  if (computer.version === latest) return { kind: "current" };
  if (!computer.automatic) return { kind: "available", version: latest };
  const tasks = activeTasks[computer.machineId] ?? 0;
  return tasks > 0 ? { kind: "waiting", version: latest, activeTasks: tasks } : { kind: "updating", version: latest };
}

export async function requestUpdate(computer: ComputerUpdates): Promise<UpdateStatus> {
  await wait(500);
  const tasks = activeTasks[computer.machineId] ?? 0;
  return tasks > 0 ? { kind: "waiting", version: latest, activeTasks: tasks } : { kind: "updating", version: latest };
}

export async function finishUpdate(): Promise<Partial<ComputerUpdates>> {
  await wait(2500);
  return { version: latest, status: { kind: "current" } };
}

export async function setAutomaticUpdates(enabled: boolean): Promise<boolean> {
  await wait(300);
  return enabled;
}
