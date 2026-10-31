// Hash router shared by the shell and the views (kept dependency-free so
// views can navigate without importing the shell).
export type Route =
  | "console"
  | "files"
  | "find"
  | "find/phone"
  | "find/requests"
  | "find/rules"
  | "sync"
  | "import"
  | "settings";

const ROUTES: Route[] = [
  "console",
  "files",
  "find",
  "find/phone",
  "find/requests",
  "find/rules",
  "sync",
  "import",
  "settings",
];

export function isRoute(value: string): value is Route {
  return (ROUTES as string[]).includes(value);
}

/** Current route from the hash (#/find/phone -> "find/phone"). */
export function currentRoute(): Route {
  const raw = window.location.hash.replace(/^#\/?/, "");
  return isRoute(raw) ? raw : "console";
}

/** Navigate to a route (hash router: Back works, the URL is shareable). */
export function navigate(route: Route): void {
  if (currentRoute() === route) {
    window.dispatchEvent(new HashChangeEvent("hashchange"));
    return;
  }
  window.location.hash = `#/${route}`;
}
