// Browser-only mock backend (vite dev without Wails). Mirrors the Go App
// method names so UI code never branches on isWails().
import type {
  ActivityRow,
  AskResult,
  DeviceRow,
  FindHit,
  MatchRule,
  RequestLogEntry,
  Status,
  StorageInfo,
  SyncEntry,
  V2State,
} from "./api";
import { mockFindHits, mockPhoneMatches, mockRequests, mockRules, mockSync } from "./mocks";
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

  // Sample `activity` + `transfer` rows so the Console table and the
  // transfer strip are reviewable without the Go backend.
  const kinds: [string, string, string][] = [
    ["file_sent", "ECOLOGY.docx · 1.69 MB", "Complete"],
    ["phone_connected", "192.168.43.21", "Connected"],
    ["file_received", "meeting-notes.pdf · 842 KB", "Complete"],
  ];
  let k = 0;
  window.setInterval(() => {
    const [kind, detail, status] = kinds[k % kinds.length];
    k += 1;
    mockActivity.push({ ts: Math.floor(Date.now() / 1000), kind, detail, status, device: "Pixel 8" });
    emit("activity", mockActivity[mockActivity.length - 1]);
  }, 12000);

  const total = 1730000;
  let sent = 0;
  window.setInterval(() => {
    sent += total / 14;
    const done = sent >= total;
    emit("transfer", {
      id: "v1-1",
      dir: "in",
      name: "ECOLOGY.docx",
      bytes: Math.min(sent, total),
      total,
      bps: 1200000,
      state: done ? "done" : "active",
      device: "Pixel 8",
    });
    if (done) sent = 0;
  }, 250);
}

function mockStatus(): Status {
  return {
    lanAddr: "192.168.43.21",
    server: {
      running: true,
      addr: "0.0.0.0:5000",
      active: 0,
      total: 3,
      bytesSent: 245000,
      bytesReceived: 1820000,
      filesSent: 32,
      filesReceived: 116,
      startTimeUnix: Date.now() / 1000 - 3600,
      uptimeSecs: 3600,
    },
    dataRoot: "D:/projects/WayerPC/data (mock)",
    shared: "D:/projects/WayerPC/data/shared (mock)",
    received: "D:/projects/WayerPC/data/received (mock)",
    onboarded: true,
    diskUsage: 2065200,
    catalogSize: mockFiles.length,
    protocolV2: true,
    device: "Pixel 8",
    controlLinks: 1,
  };
}

const mockActivity: ActivityRow[] = [
  { ts: Math.floor(Date.now() / 1000) - 90, kind: "server_started", detail: "Hotspot · 0.0.0.0:5000", status: "Listening" },
  { ts: Math.floor(Date.now() / 1000) - 40, kind: "phone_connected", detail: "192.168.43.21", status: "Connected", device: "Pixel 8" },
];

let mockMatchRules: MatchRule[] = mockRules.map((r) => ({ id: r.id, enabled: r.enabled, weight: r.weight }));
let mockSearchSeq = 0;

let mockV2: V2State = { enabled: true, protocolVersion: 2, controlLinks: 1, connectedName: "Pixel 8" };
let mockAskSeq = 0;
let mockSyncSeq = 0;
const mockLog: RequestLogEntry[] = mockRequests.map((r) => ({
  ...r,
  bundle: r.bundle?.map((b, i) => ({ n: i + 1, name: b.name, size: b.size, mtime: b.mtime })),
}));
let mockAsk: { id: string; n: number } | null = null;

const handlers: Record<string, (...args: any[]) => any> = {
  // ---- protocol v2 (phase 3) ----
  GetV2State: (): V2State => ({ ...mockV2 }),
  SetProtocolV2: (enabled: boolean): V2State => {
    mockV2 = { ...mockV2, enabled };
    return { ...mockV2 };
  },
  AskPhone: (query: string): AskResult => {
    const id = `pc-${100 + ++mockAskSeq}`;
    const q = String(query ?? "").trim();
    emit("ask-update", { id, state: "querying", query: q });
    if (!q) return { id, state: "none", query: q, elapsedMs: 12 };
    const matches = mockPhoneMatches.filter((m) => m.name.toLowerCase().includes(q.split(/\s+/)[0].toLowerCase()));
    const out = matches.length ? matches : mockPhoneMatches.slice(0, 2);
    mockAsk = { id, n: out.length };
    emit("ask-update", { id, state: "matches", query: q, matchCount: out.length });
    return { id, state: "matches", query: q, matches: out, elapsedMs: 180 };
  },
  PickAskMatch: (id: string, n: number): string => {
    if (!mockAsk || mockAsk.id !== id) throw new Error("expired: search again");
    const item = mockPhoneMatches.find((m) => m.n === n);
    emit("ask-update", { id, state: "picking", picked: n });
    emit("transfer", {
      id: `${id}-${n}`,
      dir: "in",
      name: item?.name ?? "file.bin",
      bytes: item?.size ?? 1024,
      total: item?.size ?? 1024,
      bps: 900000,
      state: "done",
      device: "Pixel 8",
    });
    const entry: RequestLogEntry = {
      id,
      ts: Math.floor(Date.now() / 1000),
      device: "Pixel 8",
      query: "mock ask",
      matchCount: mockPhoneMatches.length,
      picked: n,
      pickedName: item?.name,
      outcome: "sent",
      bundle: mockPhoneMatches.map((m) => ({ n: m.n, name: m.name, size: m.size, mtime: m.mtime })),
    };
    mockLog.push(entry);
    emit("request-update", entry);
    emit("ask-update", { id, state: "sent", picked: n, name: item?.name, path: `D:/WayerPC/data/received/${item?.name ?? "file.bin"}` });
    return `D:/WayerPC/data/received/${item?.name ?? "file.bin"}`;
  },
  AbortAsk: (id: string) => {
    emit("ask-update", { id, state: "aborted" });
    return undefined;
  },
  SendToPhone: (path: string) => {
    const name = String(path ?? "").split(/[\\/]/).pop() || "file";
    emit("ask-update", { state: "sent", name, path });
    emit("transfer", { id: `out-${++mockAskSeq}`, dir: "out", name, bytes: 120000, total: 120000, bps: 800000, state: "done", device: "Pixel 8" });
    return undefined;
  },
  // ---- sync (phase 6) ----
  StartSyncScan: () => {
    const runId = `sync-${++mockSyncSeq}`;
    const rows: SyncEntry[] = [
      ...mockSync.onlyPc.map<SyncEntry>((r) => ({
        name: r.name,
        verdict: "only_pc",
        action: "push",
        pc: { name: r.name, size: r.size, mtime: r.pcMtime },
      })),
      ...mockSync.onlyPhone.map<SyncEntry>((r) => ({
        name: r.name,
        verdict: "only_phone",
        action: "pull",
        phone: { name: r.name, size: r.size, mtime: r.phoneMtime },
      })),
      ...mockSync.different.map<SyncEntry>((r) => ({
        name: r.name,
        verdict: r.newer === "pc" ? "newer_pc" : "newer_phone",
        action: r.newer === "pc" ? "push" : "pull",
        pc: { name: r.name, size: r.size, mtime: r.pcMtime },
        phone: { name: r.name, size: r.size, mtime: r.phoneMtime },
        diffSecs: Math.abs(r.pcMtime - r.phoneMtime),
      })),
      { name: "ECOLOGY-notes.txt", verdict: "same", action: "none" },
      { name: "budget-2026.xlsx", verdict: "same", action: "none" },
      { name: "scan-0001.pdf", verdict: "same", action: "none" },
    ];
    window.setTimeout(
      () =>
        emit("sync-result", {
          runId,
          rows,
          scannedAt: Math.floor(Date.now() / 1000),
          onlyPc: mockSync.onlyPc.length,
          onlyPhone: mockSync.onlyPhone.length,
          different: mockSync.different.length,
          same: mockSync.sameCount,
        }),
      220,
    );
    return runId;
  },
  ApplySync: (names: string[]) => {
    const runId = `sync-${++mockSyncSeq}`;
    const list = Array.isArray(names) ? names : [];
    list.forEach((name, i) => {
      window.setTimeout(() => {
        emit("sync-progress", { runId, name, action: "push", state: "active", index: i + 1, total: list.length });
      }, 200 + i * 400);
      window.setTimeout(() => {
        emit("sync-progress", { runId, name, action: "push", state: "done", index: i + 1, total: list.length });
        emit("transfer", { id: `${runId}-${i}`, dir: "out", name, bytes: 120000, total: 120000, bps: 900000, state: "done", device: "Pixel 8" });
      }, 500 + i * 400);
    });
    window.setTimeout(() => handlers.StartSyncScan(), 600 + list.length * 400);
    return runId;
  },
  GetRequestLog: (): RequestLogEntry[] => mockLog.map((r) => ({ ...r })),
  GetDevices: (): DeviceRow[] => [
    { did: "did-pixel-8", name: "Pixel 8", lastSeen: Math.floor(Date.now() / 1000) - 30, lastIp: "192.168.43.21", connected: true },
    { did: "did-galaxy-a14", name: "Galaxy A14", lastSeen: Math.floor(Date.now() / 1000) - 86400 * 2, lastIp: "192.168.43.77", connected: false },
  ],
  StartFind: (query: string) => {
    const searchId = `find-${++mockSearchSeq}`;
    const tokens = String(query ?? "").toLowerCase().split(/\s+/).filter(Boolean);
    const hits: FindHit[] = mockFindHits.filter((h) => {
      const hay = `${h.name} ${h.path}`.toLowerCase();
      return tokens.every((t) => hay.includes(t)) || tokens.some((t) => hay.includes(t));
    });
    window.setTimeout(() => emit("find-hits", { searchId, hits }), 160);
    window.setTimeout(() => emit("find-done", { searchId, total: hits.length, tookMs: 42 }), 260);
    return searchId;
  },
  CancelFind: () => true,
  GetMatchRules: () => mockMatchRules.map((r) => ({ ...r })),
  SetMatchRules: (rules: MatchRule[]) => {
    mockMatchRules = rules.map((r) => ({ ...r }));
    return undefined;
  },
  ResetMatchRules: () => {
    mockMatchRules = mockRules.map((r) => ({ id: r.id, enabled: r.enabled, weight: r.weight }));
    return undefined;
  },
  GetIndexState: () => ({
    count: 12480,
    builtAt: Math.floor(Date.now() / 1000) - 3600,
    building: false,
    progress: 1,
  }),
  RebuildIndex: () => {
    emit("index-state", { count: 12480, builtAt: Math.floor(Date.now() / 1000) - 3600, building: true, progress: 0.4 });
    return "rebuilding";
  },
  GetVersion: () => ({
    appName: "WayerPC",
    version: APP_VERSION,
    protocolVersion: 2,
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
  GetActivity: () => mockActivity.slice(),
  ClearActivity: () => {
    mockActivity.length = 0;
    return undefined;
  },
  CancelTransfer: () => true,
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
