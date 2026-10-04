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
  startTimeUnix: number;
  uptimeSecs: number;
}

export interface Status {
  server: ServerStats;
  dataRoot: string;
  shared: string;
  received: string;
  onboarded: boolean;
  diskUsage: number;
  catalogSize: number;
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
  getSetting: (key: string) => call<string>("GetSetting", key),
  setSetting: (key: string, value: string) => call<void>("SetSetting", key, value),
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
