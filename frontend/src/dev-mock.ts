// Browser-only mock backend (vite dev without Wails). Mirrors the Go App
// method names so UI code never branches on isWails().
import type { Status, StorageInfo } from "./api";
import { APP_VERSION } from "./version";

type Handler = (data: any) => void;

const subs = new Map<string, Set<Handler>>();
let started = false;

const mockFiles = [
  { name: "report.pdf", path: "D:/projects/docs/report.pdf", size: 245000, modTime: new Date().toISOString(), isDir: false },
  { name: "summer-photo.jpg", path: "D:/projects/photos/summer-photo.jpg", size: 1820000, modTime: new Date().toISOString(), isDir: false },
  { name: "notes.txt", path: "D:/projects/notes.txt", size: 1200, modTime: new Date().toISOString(), isDir: false },
];

function emit(event: string, data: any = null): void {
  subs.get(event)?.forEach((h) => {
    try {
      h(data);
    } catch {
      /* mock must never break the UI */
    }
  });
}

function boot(): void {
  if (started) return;
  started = true;
  let n = 0;
  window.setInterval(() => {
    n += 1;
    emit("log", `[MOCK] heartbeat ${n} — run under Wails for live data`);
    emit("stats", mockStatus());
  }, 5000);
}

function mockStatus(): Status {
  return {
    server: {
      running: true,
      addr: "0.0.0.0:5000",
      active: 0,
      total: 3,
      bytesSent: 245000,
      bytesReceived: 1820000,
      startTimeUnix: Date.now() / 1000 - 3600,
      uptimeSecs: 3600,
    },
    dataRoot: "D:/projects/WayerPC/data (mock)",
    shared: "D:/projects/WayerPC/data/shared (mock)",
    received: "D:/projects/WayerPC/data/received (mock)",
    onboarded: true,
    diskUsage: 2065200,
    catalogSize: mockFiles.length,
  };
}

const handlers: Record<string, (...args: any[]) => any> = {
  GetVersion: () => ({
    appName: "WayerPC",
    version: APP_VERSION,
    protocolVersion: 1,
    description: "Phone ↔ PC transfer over hotspot (mock)",
  }),
  GetStatus: () => mockStatus(),
  StartServer: () => "starting (mock)",
  StopServer: () => "stopped (mock)",
  GetLogs: () => ["[MOCK] browser preview — logs stream here under Wails"],
  ClearLogs: () => undefined,
  SearchUser: (q: string) =>
    mockFiles.filter((f) => f.name.toLowerCase().includes(String(q).toLowerCase())),
  SearchApp: (q: string) =>
    mockFiles.filter((f) => f.name.toLowerCase() === String(q).toLowerCase()),
  CatalogList: () =>
    mockFiles.map((f, i) => ({ id: i + 1, path: f.path, name: f.name, sizeBytes: f.size, importedAt: f.modTime, exists: true })),
  CatalogAddPaths: () => ({ added: 1, skipped: 0 }),
  CatalogAddFolder: () => ({ added: 2, skipped: 0 }),
  CatalogRemove: () => undefined,
  ListReceived: () => [mockFiles[1]],
  ListShared: () => [],
  GetStorageInfo: (): StorageInfo => ({
    dataRoot: "D:/projects/WayerPC/data (mock)",
    installDir: "D:/projects/WayerPC",
    shared: "D:/projects/WayerPC/data/shared (mock)",
    received: "D:/projects/WayerPC/data/received (mock)",
    dbPath: "D:/projects/WayerPC/data/Data/wayerpc.db (mock)",
    diskUsage: 2065200,
    onboarded: true,
  }),
  SetDataRoot: (root: string) => ({ ...handlers.GetStorageInfo(), dataRoot: root }),
  PickFolder: () => "",
  PickFiles: () => null,
  RevealInExplorer: () => "ok (mock)",
  GetSetting: () => "",
  SetSetting: () => undefined,
};

export async function mockCall<T>(method: string, ...args: any[]): Promise<T> {
  boot();
  const h = handlers[method];
  if (!h) throw new Error(`mock: unknown method ${method}`);
  await new Promise((r) => window.setTimeout(r, 120)); // feel the latency
  return h(...args) as T;
}

export function mockOn(event: string, cb: Handler): () => void {
  boot();
  let set = subs.get(event);
  if (!set) {
    set = new Set();
    subs.set(event, set);
  }
  set.add(cb);
  return () => {
    subs.get(event)?.delete(cb);
  };
}
