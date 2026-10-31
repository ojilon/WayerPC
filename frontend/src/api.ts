// Typed bridge to the Go backend. In the Wails window every call hits
// window.go.main.App.* and events arrive via window.runtime.EventsOn.
// In a plain browser (vite dev) the mock backend answers instead, so the
// UI is fully workable without Go.

declare global {
  interface Window {
    go?: any;
    runtime?: any;
  }
}

export interface VersionInfo {
  appName: string;
  version: string;
  protocolVersion: number;
  description: string;
}

export interface ServerStats {
  running: boolean;
  addr: string;
  active: number;
  total: number;
  bytesSent: number;
  bytesReceived: number;
  filesSent: number;
  filesReceived: number;
  startTimeUnix: number;
  uptimeSecs: number;
}

export interface ActivityRow {
  ts: number;
  kind: string;
  detail: string;
  status: string;
  device?: string;
}

export type TransferState = "active" | "done" | "error" | "cancelled";

export interface TransferRow {
  id: string;
  dir: "in" | "out";
  name: string;
  bytes: number;
  total: number;
  bps: number;
  state: TransferState;
  device?: string;
}

export interface Status {
  server: ServerStats;
  /** Private IPv4 a phone can dial ("" until an interface has one). */
  lanAddr: string;
  dataRoot: string;
  shared: string;
  received: string;
  onboarded: boolean;
  diskUsage: number;
  catalogSize: number;
  /** Protocol v2 opt-in state + live control link (phase 3). */
  protocolV2?: boolean;
  device?: string;
  controlLinks?: number;
}

export interface StorageInfo {
  dataRoot: string;
  installDir: string;
  shared: string;
  received: string;
  dbPath: string;
  diskUsage: number;
  onboarded: boolean;
}

export interface ImportResult {
  added: number;
  skipped: number;
}

export interface SearchHit {
  name: string;
  path: string;
}

/** One ranked Find hit (internal/rank.Hit). */
export interface FindHit {
  path: string;
  name: string;
  dir?: string;
  ext?: string;
  size: number;
  mtime: number;
  score: number;
  reason: string;
}

export interface FindOpts {
  limit?: number;
  cutoff?: number;
}

export interface FindHitsEvent {
  searchId: string;
  hits: FindHit[];
}

export interface FindDoneEvent {
  searchId: string;
  total: number;
  tookMs: number;
  skippedNote?: string;
}

/** One search rule configuration (internal/rank.Config). */
export interface MatchRule {
  id: string;
  enabled: boolean;
  weight: number;
}

export interface IndexState {
  count: number;
  builtAt: number;
  building: boolean;
  progress: number;
  note?: string;
}

export interface CatalogEntry {
  id: number;
  path: string;
  name: string;
  sizeBytes: number;
  importedAt: string;
  score?: number;
  exists?: boolean;
}

export interface FileItem {
  name: string;
  path: string;
  size: number;
  modTime: string;
  isDir: boolean;
}


/** Protocol-v2 status (Settings + title bar). */
export interface V2State {
  enabled: boolean;
  protocolVersion: number;
  controlLinks: number;
  connectedName?: string;
}

/** One numbered match offered by the phone (ask-phone flow). */
export interface PhoneItem {
  n: number;
  name: string;
  size: number;
  mtime: number;
}

export interface AskResult {
  id: string;
  state: "matches" | "none" | "error";
  query: string;
  matches?: PhoneItem[];
  error?: string;
  device?: string;
  elapsedMs: number;
}

/** `ask-update` event payload. */
export interface AskUpdate {
  id?: string;
  state: "querying" | "matches" | "none" | "picking" | "sent" | "error" | "aborted";
  query?: string;
  matchCount?: number;
  picked?: number;
  name?: string;
  path?: string;
  error?: string;
}

/** One row of the phone-initiated request log. */
export interface RequestLogEntry {
  id: string;
  ts: number;
  device: string;
  query: string;
  matchCount: number;
  picked?: number;
  pickedName?: string;
  outcome: "sent" | "no-match" | "aborted" | "error" | "pending";
  bundle?: PhoneItem[];
}

/** One row of the Sync comparison (internal/syncplan.Entry). */
export interface SyncEntry {
  name: string;
  verdict: "only_pc" | "only_phone" | "newer_pc" | "newer_phone" | "same";
  action: "push" | "pull" | "none";
  pc?: { name: string; size: number; mtime: number; sha256?: string };
  phone?: { name: string; size: number; mtime: number; sha256?: string };
  diffSecs?: number;
}

/** `sync-result` event payload. */
export interface SyncResult {
  runId: string;
  rows: SyncEntry[];
  note?: string;
  error?: string;
  scannedAt: number;
  onlyPc?: number;
  onlyPhone?: number;
  different?: number;
  same?: number;
}

/** `sync-progress` event payload. */
export interface SyncProgress {
  runId: string;
  name: string;
  action: "push" | "pull";
  state: "queued" | "active" | "done" | "error" | "skipped";
  index: number;
  total: number;
  error?: string;
}

export interface DeviceRow {
  did: string;
  name: string;
  lastSeen: number;
  lastIp: string;
  connected: boolean;
}
export interface Ctx {
  toast: (msg: string) => void;
  confirm: (msg: string) => Promise<boolean>;
}

function bound(): any | null {
  const app = window.go?.main?.App;
  return app ?? null;
}

export function isWails(): boolean {
  return bound() !== null;
}

async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  const b = bound();
  if (b && typeof b[method] === "function") {
    return (await b[method](...args)) as T;
  }
  const { mockCall } = await import("./dev-mock");
  return mockCall<T>(method, ...args);
}

/** Subscribe to a backend event. Returns an unsubscribe function. */
export function onEvent(event: string, cb: (data: any) => void): () => void {
  const rt = window.runtime;
  if (rt?.EventsOn) {
    rt.EventsOn(event, cb);
    return () => rt.EventsOff?.(event);
  }
  let off = () => {};
  import("./dev-mock").then((m) => {
    off = m.mockOn(event, cb);
  });
  return () => off();
}

export const api = {
  getVersion: () => call<VersionInfo>("GetVersion"),
  getStatus: () => call<Status>("GetStatus"),
  startServer: () => call<string>("StartServer"),
  stopServer: () => call<string>("StopServer"),
  getLogs: () => call<string[]>("GetLogs"),
  clearLogs: () => call<void>("ClearLogs"),
  searchUser: (query: string, limit = 200) => call<SearchHit[]>("SearchUser", query, limit),
  searchApp: (query: string) => call<SearchHit[]>("SearchApp", query),
  catalogList: () => call<CatalogEntry[]>("CatalogList"),
  catalogAddPaths: (paths: string[]) => call<ImportResult>("CatalogAddPaths", paths),
  catalogAddFolder: (folder: string) => call<ImportResult>("CatalogAddFolder", folder),
  catalogRemove: (path: string) => call<void>("CatalogRemove", path),
  listReceived: () => call<FileItem[]>("ListReceived"),
  listShared: () => call<FileItem[]>("ListShared"),
  getStorageInfo: () => call<StorageInfo>("GetStorageInfo"),
  setDataRoot: (root: string) => call<StorageInfo>("SetDataRoot", root),
  pickFolder: (title: string) => call<string>("PickFolder", title),
  pickFiles: () => call<string[] | null>("PickFiles"),
  reveal: (path: string) => call<string>("RevealInExplorer", path),
  startFind: (query: string, opts: FindOpts = {}) => call<string>("StartFind", query, opts),
  cancelFind: (searchId: string) => call<boolean>("CancelFind", searchId),
  getMatchRules: () => call<MatchRule[]>("GetMatchRules"),
  setMatchRules: (rules: MatchRule[]) => call<void>("SetMatchRules", rules),
  resetMatchRules: () => call<void>("ResetMatchRules"),
  getIndexState: () => call<IndexState>("GetIndexState"),
  rebuildIndex: () => call<string>("RebuildIndex"),
  getActivity: () => call<ActivityRow[]>("GetActivity"),
  clearActivity: () => call<void>("ClearActivity"),
  cancelTransfer: (id: string) => call<boolean>("CancelTransfer", id),
  getSetting: (key: string) => call<string>("GetSetting", key),
  setSetting: (key: string, value: string) => call<void>("SetSetting", key, value),
  // ---- protocol v2 (phase 3) ----
  getV2State: () => call<V2State>("GetV2State"),
  setProtocolV2: (enabled: boolean) => call<V2State>("SetProtocolV2", enabled),
  askPhone: (query: string) => call<AskResult>("AskPhone", query),
  pickAskMatch: (id: string, n: number) => call<string>("PickAskMatch", id, n),
  abortAsk: (id: string) => call<void>("AbortAsk", id),
  sendToPhone: (path: string) => call<void>("SendToPhone", path),
  getRequestLog: () => call<RequestLogEntry[]>("GetRequestLog"),
  getDevices: () => call<DeviceRow[]>("GetDevices"),
  // ---- sync (phase 6) ----
  startSyncScan: () => call<string>("StartSyncScan"),
  applySync: (names: string[]) => call<string>("ApplySync", names),
};

export const win = {
  min: () => window.runtime?.WindowMinimise?.(),
  toggleMax: () => window.runtime?.WindowToggleMaximise?.(),
  quit: () => window.runtime?.Quit?.(),
};

export function fmtMB(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export function fmtDate(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}
