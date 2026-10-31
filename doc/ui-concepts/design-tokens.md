# WayerPC UI design tokens

A quiet, ruled, typographic dark interface. Group related information through whitespace and fine dividers rather than cards. Use the blue accent for active navigation, links, selected states, and primary actions. Status colors are reserved for small dots and short status labels.

## Color

| Token | Hex | Use |
|---|---|---|
| Background (`bg`) | `#111416` | Main application canvas |
| Surface (`surface`) | `#171B1E` | Sidebar and quiet utility regions |
| Raised surface (`surface-2`) | `#1C2125` | Subtle secondary surface only |
| Border (`border`) | `#30373C` | 1px hairlines, separators, and dividers |
| Text (`text`) | `#E5E9EC` | Primary labels and content |
| Muted (`muted`) | `#929CA4` | Metadata, helper copy, inactive navigation |
| Faint (`faint`) | `#667179` | Low-priority symbols and disabled detail |
| Accent (`accent`) | `#6AB8F2` | Active item, links, selection, progress, primary action |
| Success (`green`) | `#65C98E` | Small connected/listening dots and success labels |
| Warning (`amber`) | `#E0AD63` | Small pending/warning indicator |
| Error (`red`) | `#E77C7C` | Error indicator, Exit action, short error text |

## Typography

| Role | Typeface | Size / weight |
|---|---|---|
| Main UI | Inter, with Segoe UI / Arial fallback | 13–14px / 400–600 |
| Screen title | Main UI | 24px / 560 |
| Section label | Main UI, uppercase with letter spacing | 10px / 650 |
| Supporting copy | Main UI | 11–12px / 400 |
| Monospace data | SFMono-Regular, Consolas, Liberation Mono | 10–12px / 400 |
| Primary action | Main UI | 13px / 600 |

## Layout and control dimensions

| Token | Value |
|---|---|
| Canvas | 1366 × 768px |
| Window title bar | 44px high |
| Expanded sidebar | 208px wide |
| Collapsed sidebar | 56px wide |
| Main content inset | 40px horizontal; 28px top |
| Sidebar navigation row | 41px high |
| File/result list row | 44px target baseline; multi-line file rows expand to fit path and metadata |
| Global transfer strip | 50px high |
| Spacing scale | 4, 8, 12, 16, 20, 28, 40px |
| Standard border | 1px solid `border` |
| Active tab / sidebar indicator | 2px accent rule |
| Input treatment | Transparent, bottom border only |
| Default button treatment | Plain text link; one filled primary action at most per screen |
| Corner radius | 0–2px only; no cards or rounded panels |

## Interaction styling

Navigation selection uses accent text and a 2px left-edge rule. Tabs use an accent underline, not a pill. Text actions have no button chrome and underline on hover. File, result, and log rows are separated by hairlines. Inputs are transparent with a bottom border. Checkboxes, sliders, and drag handles are drawn as simple CSS shapes; no image assets are required.
