// Inline SVG icons (no image assets). 1.5px strokes on a 24px grid, drawn
// with currentColor so a row can tint them by state.

const PATHS: Record<string, string> = {
  console:
    '<path d="M4 6.5h16M4 12h16M4 17.5h16"/><circle cx="2.6" cy="6.5" r=".9" fill="currentColor" stroke="none"/>',
  files: '<path d="M3.5 7.5h6l1.8 2h9.2v9.5a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1V8.5a1 1 0 0 1 1-1Z"/><path d="M3.5 12.5h17"/>',
  find: '<circle cx="10.5" cy="10.5" r="6"/><path d="m15 15 5 5"/>',
  sync: '<path d="M4 8.5h13l-3-3M20 15.5H7l3 3"/>',
  import: '<path d="M12 3v12m0 0 4.5-4.5M12 15 7.5 10.5"/><path d="M4 17v2.5a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V17"/>',
  settings:
    '<circle cx="12" cy="12" r="3"/><path d="M12 3.5v2M12 18.5v2M4.6 7.5l1.8 1M17.6 15.5l1.8 1M4.6 16.5l1.8-1M17.6 8.5l1.8-1"/>',
  exit: '<path d="M14 4.5H6.5a1 1 0 0 0-1 1v13a1 1 0 0 0 1 1H14"/><path d="M17 8.5 20.5 12 17 15.5M10 12h10.5"/>',
  chevronLeft: '<path d="M14 6 8 12l6 6"/>',
  chevronRight: '<path d="M10 6l6 6-6 6"/>',
  arrowRight: '<path d="M4 12h15m0 0-5-5m5 5-5 5"/>',
  arrowLeft: '<path d="M20 12H5m0 0 5-5m-5 5 5 5"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7"/>',
  dot: '<circle cx="12" cy="12" r="4" fill="currentColor" stroke="none"/>',
  drag: '<path d="M9 6h.01M9 12h.01M9 18h.01M15 6h.01M15 12h.01M15 18h.01" stroke-width="2.4"/>',
};

export function icon(name: keyof typeof PATHS | string, size = 15): string {
  const body = PATHS[name] ?? PATHS.dot;
  return `<svg class="icon" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none"
    stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"
    aria-hidden="true">${body}</svg>`;
}
