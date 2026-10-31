// Tiny DOM helpers shared by the views. Deliberately framework-free: no
// virtual DOM, no per-second tree walks — views build their rows once and
// patch the few nodes that change.

/** Escape text for safe interpolation into innerHTML. */
export function esc(value: unknown): string {
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/** querySelector that throws instead of returning null (template typos fail fast). */
export function q<T extends HTMLElement = HTMLElement>(root: ParentNode, sel: string): T {
  const el = root.querySelector(sel);
  if (!el) throw new Error(`missing element: ${sel}`);
  return el as T;
}

export function qa<T extends HTMLElement = HTMLElement>(root: ParentNode, sel: string): T[] {
  return Array.from(root.querySelectorAll(sel)) as T[];
}

/** Format bytes the way the concepts do: "1.73 MB", "842 KB", "12 KB". */
export function fmtBytes(bytes: number): string {
  const n = Number(bytes) || 0;
  if (n < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = n / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const digits = value >= 100 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(digits).replace(/\.?0+$/, "")} ${units[unit]}`;
}

/** "1.2 MB/s" for the transfer strip. */
export function fmtRate(bytesPerSec: number): string {
  return `${fmtBytes(bytesPerSec)}/s`;
}

/** HH:MM:SS for the activity table. */
export function fmtClock(unixSeconds: number): string {
  const d = new Date((unixSeconds || 0) * 1000);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
}

/** "10/28/2026 8:17 AM" style short date+time. */
export function fmtDateTime(iso: string | number | undefined): string {
  if (iso === undefined || iso === "" || iso === 0) return "—";
  const d = typeof iso === "number" ? new Date(iso * 1000) : new Date(iso);
  if (Number.isNaN(d.getTime())) return String(iso);
  return d.toLocaleString([], { year: "numeric", month: "numeric", day: "numeric", hour: "numeric", minute: "2-digit" });
}

/** "10/28/2026" only. */
export function fmtDate(iso: string | number | undefined): string {
  if (iso === undefined || iso === "" || iso === 0) return "—";
  const d = typeof iso === "number" ? new Date(iso * 1000) : new Date(iso);
  if (Number.isNaN(d.getTime())) return String(iso);
  return d.toLocaleDateString();
}

/** 02:14:38 for uptime. */
export function fmtUptime(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds || 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  return [h, m, sec].map((v) => String(v).padStart(2, "0")).join(":");
}

/** Group list items into a "show more" stepper (100 rows per chunk). */
export const PAGE_SIZE = 100;

/** Percent 0..100, safe for divide-by-zero. */
export function pct(done: number, total: number): number {
  if (!total || total <= 0) return 0;
  return Math.max(0, Math.min(100, Math.round((done / total) * 100)));
}

/** Human labels for activity kinds (Console EVENT column). */
export const ACTIVITY_LABELS: Record<string, string> = {
  server_started: "Server started",
  server_stopped: "Server stopped",
  phone_connected: "Phone connected",
  phone_disconnected: "Phone disconnected",
  file_sent: "File sent",
  file_received: "File received",
  matches_sent: "Matches sent",
  import: "Import",
  error: "Error",
};

export function activityLabel(kind: string): string {
  return ACTIVITY_LABELS[kind] ?? kind.replace(/_/g, " ");
}

/** Only http(s)/file-ish paths are rendered as links; everything else is text. */
export function isSafeHref(href: string): boolean {
  return /^(https?:|#|\/)/i.test(href);
}
