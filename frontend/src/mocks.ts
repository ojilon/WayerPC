// Static sample data for the screens whose backend wiring lands in later
// phases (see the TODO(phaseN) markers in each pane). Kept in one place so
// the mock shapes match the eventual bridge payloads 1:1 and dev-mock.ts can
// reuse them.

export interface FindHit {
  path: string;
  name: string;
  size: number;
  mtime: number;
  score: number;
  reason: string;
}

export interface PhoneMatch {
  n: number;
  name: string;
  size: number;
  mtime: number;
}

export interface RequestEntry {
  id: string;
  ts: number;
  device: string;
  query: string;
  matchCount: number;
  picked?: number;
  pickedName?: string;
  outcome: "sent" | "no-match" | "aborted";
  bundle?: { name: string; size: number; mtime: number }[];
}

export interface RuleRow {
  id: string;
  label: string;
  enabled: boolean;
  weight: number;
}

export interface SyncRow {
  name: string;
  size: number;
  pcMtime: number;
  phoneMtime: number;
  newer: "pc" | "phone";
}

const now = Math.floor(Date.now() / 1000);

export const mockFindHits: FindHit[] = [
  {
    path: "D:/WayerPC/data/received/ECOLOGY.docx",
    name: "ECOLOGY.docx",
    size: 1730000,
    mtime: now - 86400 * 3,
    score: 0.92,
    reason: "exact name",
  },
  {
    path: "D:/Documents/Research/Ecology_2026_final.pdf",
    name: "Ecology_2026_final.pdf",
    size: 2080000,
    mtime: now - 86400 * 9,
    score: 0.81,
    reason: "contains",
  },
  {
    path: "D:/Archive/Reports/report-ecology-notes.txt",
    name: "report-ecology-notes.txt",
    size: 18000,
    mtime: now - 86400 * 13,
    score: 0.64,
    reason: "same folder",
  },
];

export const mockPhoneMatches: PhoneMatch[] = [
  { n: 1, name: "Ecology Report 2026.pdf", size: 4404019, mtime: now - 86400 * 2 },
  { n: 2, name: "ecology-final.docx", size: 1887436, mtime: now - 86400 * 9 },
  { n: 3, name: "ECOLOGY.docx", size: 1730000, mtime: now - 86400 * 11 },
];

export const mockRequests: RequestEntry[] = [
  {
    id: "#024",
    ts: now - 900,
    device: "Pixel 8",
    query: "ecology report",
    matchCount: 3,
    picked: 2,
    pickedName: "ecology-final.docx",
    outcome: "sent",
    bundle: [
      { name: "Ecology Report 2026.pdf", size: 4404019, mtime: now - 86400 * 2 },
      { name: "ecology-final.docx", size: 1887436, mtime: now - 86400 * 9 },
      { name: "ECOLOGY.docx", size: 1730000, mtime: now - 86400 * 11 },
    ],
  },
  { id: "#023", ts: now - 5000, device: "Pixel 8", query: "meeting notes", matchCount: 1, outcome: "sent" },
  { id: "#022", ts: now - 8900, device: "Pixel 8", query: "budget", matchCount: 0, outcome: "no-match" },
];

export const mockRules: RuleRow[] = [
  { id: "exact_name", label: "Exact name", enabled: true, weight: 100 },
  { id: "name_contains", label: "Name contains", enabled: true, weight: 78 },
  { id: "extension", label: "Extension", enabled: true, weight: 55 },
  { id: "recently_modified", label: "Recently modified", enabled: true, weight: 42 },
  { id: "fuzzy_name", label: "Fuzzy name", enabled: true, weight: 36 },
  { id: "folder_hint", label: "Folder hint", enabled: true, weight: 28 },
];

export const mockSync = {
  onlyPc: [
    { name: "proposal-v3.docx", size: 1258291, pcMtime: now - 86400, phoneMtime: 0 },
    { name: "notes-oct.txt", size: 24576, pcMtime: now - 86400 * 4, phoneMtime: 0 },
  ] as SyncRow[],
  onlyPhone: [
    { name: "IMG_4821.jpg", size: 3250000, pcMtime: 0, phoneMtime: now - 86400 * 2 },
    { name: "voice-note.m4a", size: 880000, pcMtime: 0, phoneMtime: now - 86400 * 6 },
  ] as SyncRow[],
  different: [
    { name: "ECOLOGY.docx", size: 1730000, pcMtime: now - 86400 * 3, phoneMtime: now - 86400 * 5, newer: "pc" },
    { name: "trip-plan.pdf", size: 640000, pcMtime: now - 86400 * 8, phoneMtime: now - 86400, newer: "phone" },
  ] as SyncRow[],
  sameCount: 48,
  lastScan: now - 240,
};
