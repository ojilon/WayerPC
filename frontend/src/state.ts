// Shared, tiny app state: status, structured activity and live transfers.
// One set of backend subscriptions feeds every screen; screens subscribe to
// this store instead of each opening its own event handlers. No polling and
// no timers here — the backend pushes `stats` once a second.
import {
  api,
  onEvent,
  type ActivityRow,
  type AskUpdate,
  type RequestLogEntry,
  type Status,
  type TransferRow,
  type V2State,
} from "./api";

type Listener = () => void;

export interface LinkState {
  tone: "green" | "amber" | "red";
  label: string;
}

const listeners = new Set<Listener>();

export const store = {
  status: null as Status | null,
  activity: [] as ActivityRow[],
  /** Keyed by transfer id so progress updates replace their row. */
  transfers: new Map<string, TransferRow>(),
  versionInfo: null as { version: string; protocolVersion: number } | null,
  /** Protocol v2: opt-in state + live control link. */
  v2: null as V2State | null,
  /** Latest `ask-update` (Ask phone stepper drives itself from this). */
  ask: null as AskUpdate | null,
  /** Phone-initiated request log, oldest first. */
  requests: [] as RequestLogEntry[],
};

let started = false;

function notify(): void {
  listeners.forEach((fn) => {
    try {
      fn();
    } catch (err) {
      console.error("listener failed", err);
    }
  });
}

/** Subscribe to store changes. Returns an unsubscribe function. */
export function listen(fn: Listener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

/** Start the backend subscriptions (idempotent). */
export async function initState(): Promise<void> {
  if (started) return;
  started = true;

  onEvent("stats", (st: Status) => {
    store.status = st;
    notify();
  });
  onEvent("activity", (row: ActivityRow) => {
    if (!row) return;
    store.activity = [...store.activity, row].slice(-2000);
    notify();
  });
  onEvent("transfer", (t: TransferRow) => {
    if (!t || !t.id) return;
    store.transfers.set(t.id, t);
    if (t.state !== "active") {
      // Keep the finished row visible for a moment, then drop it so the
      // strip disappears (it is only shown while transfers are active).
      window.setTimeout(() => {
        const row = store.transfers.get(t.id);
        if (row && row.state !== "active") {
          store.transfers.delete(t.id);
          notify();
        }
      }, 4000);
    }
    notify();
  });

  onEvent("ask-update", (u: AskUpdate) => {
    if (!u || !u.state) return;
    store.ask = u;
    notify();
  });
  onEvent("request-update", (row: RequestLogEntry) => {
    if (!row || !row.id) return;
    const i = store.requests.findIndex((r) => r.id === row.id);
    if (i >= 0) store.requests = store.requests.map((r) => (r.id === row.id ? row : r));
    else store.requests = [...store.requests, row].slice(-200);
    notify();
  });

  try {
    const [status, activity, version, v2, requests] = await Promise.all([
      api.getStatus(),
      api.getActivity(),
      api.getVersion(),
      api.getV2State(),
      api.getRequestLog(),
    ]);
    store.status = status;
    store.activity = activity.slice(-2000);
    store.versionInfo = { version: version.version, protocolVersion: version.protocolVersion };
    store.v2 = v2;
    store.requests = requests ?? [];
  } catch {
    // A missing backend (browser mock) still renders; screens show states.
  }
  notify();
}

/** Refresh the status snapshot once (used after actions that change it). */
export async function refreshStatus(): Promise<Status | null> {
  try {
    store.status = await api.getStatus();
    notify();
    return store.status;
  } catch {
    return store.status;
  }
}

/** Replace the activity buffer (Clear). */
export async function clearActivity(): Promise<void> {
  store.activity = [];
  notify();
  try {
    await api.clearActivity();
  } catch {
    /* mock */
  }
}

/** Active transfers, newest first. */
export function activeTransfers(): TransferRow[] {
  const rows = Array.from(store.transfers.values()).filter((t) => t.state === "active");
  return rows.reverse();
}

/** Refresh the v2 connection snapshot (after a call that changes it). */
export async function refreshV2(): Promise<V2State | null> {
  try {
    store.v2 = await api.getV2State();
    notify();
  } catch {
    /* mock */
  }
  return store.v2;
}

/** Reload the request log (after an ask/pick or on pane mount). */
export async function refreshRequests(): Promise<RequestLogEntry[]> {
  try {
    store.requests = (await api.getRequestLog()) ?? [];
    notify();
  } catch {
    /* mock */
  }
  return store.requests;
}

/** True when a phone currently holds a v2 control link. */
export function phoneLinked(status: Status | null = store.status): boolean {
  const st = status ?? store.status;
  if (st?.device) return true;
  return (store.v2?.controlLinks ?? 0) > 0;
}

/** Titlebar connection indicator (green / amber / red). */
export function linkState(status: Status | null): LinkState {
  if (!status) return { tone: "amber", label: "Connecting…" };
  if (!status.server?.running) return { tone: "red", label: "Server stopped" };
  // Phase 5 fills `device` from the v2 control-link registry.
  const device = (status as Status & { device?: string }).device;
  if (device) return { tone: "green", label: `Connected · ${device}` };
  return { tone: "amber", label: "Listening · no phone" };
}

/** The address a phone should type in (private IPv4 + the real port). */
export function phoneAddr(status: Status | null): string {
  if (!status) return "—";
  const addr = status.server?.addr ?? "";
  const port = addr.split(":")[1] || "5000";
  const lan = (status as Status & { lanAddr?: string }).lanAddr;
  if (lan) return `${lan}:${port}`;
  const host = addr.split(":")[0];
  if (!host || host === "0.0.0.0" || host === "::") return `127.0.0.1:${port}`;
  return `${host}:${port}`;
}
