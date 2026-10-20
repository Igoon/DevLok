# Prompt Google Stitch - DevLok

Dokumen ini berisi prompt siap tempel ke Google Stitch untuk membuat alternatif UI DevLok (desktop app) dan tray flyout.

Cara pakai:

1. Buka Google Stitch.
2. Pilih mode Desktop/Web.
3. Tempel isi blok **Prompt App** untuk desain aplikasi.
4. Buat project baru atau frame terpisah, tempel **Prompt Tray** untuk flyout tray.
5. Jika Stitch mendukung multi-screen, tambahkan instruksi screen tambahan di bagian bawah.

---

## Prompt App

```text
Design a production-quality desktop developer tool UI named "DevLok". It is a local development environment manager for Windows, similar in purpose to Laragon but modern, precise, and calm.

Generate a desktop web app at 1440 x 1024 with 4 screens:
1. Dashboard
2. Configuration
3. Create Site modal
4. Import Laragon wizard

VISUAL DIRECTION
- Dark, technical, low-noise developer tool. Not a marketing landing page.
- Background: #0E1014. Surface: #15181F. Elevated surface: #1A1E27.
- Border: #272C38. Divider: #20242E.
- Text primary: #E8EDF7. Text secondary: #9AA3B5. Text muted: #6F788B.
- Accent primary: #5B8DEF (actions). Accent success: #2ED3A0 (running). Accent warning: #F0B45A (partial/starting). Accent danger: #F06A7A (failed/stop).
- Typography: Inter or Geist. 14px body, 13px table, 12px labels, 26-30px page title. Monospace: JetBrains Mono or Cascadia Code for paths, ports, URLs, versions.
- Radius: 10px controls, 14px panels, 999px status pills.
- Borders and spacing do the work. Shadows minimal. No glassmorphism, no neon glow, no purple gradient, no emoji, no oversized hero sections.
- Dense but breathable. 8px spacing grid. Clear hierarchy.
- Icon style: 1.5px stroke, rounded, consistent. Use Lucide-style icons.

APP SHELL
- Left sidebar, 248px wide, full height.
  - Top: DevLok wordmark with a small geometric mark. Subtitle "local dev".
  - Primary nav: Dashboard, Configuration, Import, Logs.
  - Section "Sites": list of projects with tiny status dot and domain. Active project highlighted with a subtle blue left border.
  - Bottom: compact runtime health summary and version "0.1.0".
- Main area:
  - Top bar: breadcrumb "Sites / coba", project domain in monospace, global running badge, command palette hint "Ctrl K", refresh icon.
  - Content background matches app background, panels separated by border, not floating cards everywhere.

SCREEN 1 - DASHBOARD
Purpose: see and control the active site in one glance.

Header block:
- Project name "coba" large, domain "coba.test" in monospace below.
- Status badge: green dot + "Running".
- Action row: primary "Open site", secondary "phpMyAdmin", "Folder", "Editor", "Terminal", danger "Stop".
- URL strip: two monospace chips "https://coba.test" and "http://127.0.0.1". Copy icon on hover.

Health summary row, 4 compact metrics:
- HTTP 80 - Listening (green text)
- HTTPS 443 - Listening (green text)
- Database coba - MySQL 8.4.10 (green text)
- DB user coba - port 3306 (monospace)
Each metric is a bordered surface, 72px tall, label uppercase 11px, value 16px semibold.

Services section:
- Section header "Services" with subtitle "Only active services are shown".
- Table columns: Service, Version, Port, Status, Uptime, Actions.
- Rows:
  1. MySQL, version dropdown "8.4.10", port 3306, green "running", uptime 00:14:32, actions: power toggle on, restart icon.
  2. Nginx, version dropdown "1.30.6", port 80/443, running, uptime, actions: power toggle on, restart.
  3. PHP-FPM, version dropdown "8.4.26", port 19000 internal, running, actions: power toggle on, restart.
- Each version is an inline select styled as a subtle chip, not a plain browser select.
- Power toggle: compact switch, green when on, gray when off.
- When a service is off, row stays but muted, status pill "stopped", toggle off.
- Include one alternate state frame: MySQL off, showing "phpMyAdmin unavailable" inline note.
- Add a small "Components" toolbar above the table: Database toggle, phpMyAdmin toggle, "Apply and restart" button. Make it look integrated, not a separate settings card.

Sites section:
- Horizontal list or 2-column grid of project cards.
- Card: project name, domain, PHP version, root path in monospace, status, one primary action.
- Active card has a thin blue outline and "active" pill. Inactive cards show only "Start" action to avoid duplicate buttons.
- Include "New site" dashed tile.

Activity / Logs panel:
- Right-side or lower split panel, 380px wide.
- Header "Live logs" with service filter tabs: nginx, php, mysql, app.
- Monospace log lines, timestamps muted, level colors, auto-scroll toggle, pause button.
- Empty state: "No log output yet".

Footer:
- Left: "DevLok 0.1.0".
- Right: last action or idle status, subtle.

SCREEN 2 - CONFIGURATION
Purpose: edit project-level settings without editing YAML.

Layout: page header + two-column form, max width 1040px, sticky bottom action bar.

Header:
- Title "Configuration".
- Subtitle "Project settings are written to .devlok.yaml and .devlok.lock".
- Project selector in top right.

Section "Virtual host":
- Domain field "coba.test".
- Aliases multi-value input chips: "www.coba.test", "api.coba.test", with add and remove.
- HTTPS toggle.
- Show resolved URL preview below.

Section "Ports":
- 2x2 grid: HTTP 80, HTTPS 443, MySQL 3306, PHP FastCGI 19000.
- Each input has helper text "0 = automatic".
- Show a small conflict warning variant for an occupied port, amber inline message.

Section "Service versions":
- Four rows: Nginx, PHP, MySQL, Node.js.
- Each row: service icon, current version dropdown, installed badge, "latest" hint.
- Dropdown options show exact versions.
- Node.js has "Not used" option.
- Add a subtle warning when changing MySQL version: "Uses a separate data directory. Existing data is not migrated automatically."

Section "Components":
- Database switch with description "MySQL and project database".
- phpMyAdmin switch, disabled when database is off, with helper text.

Sticky action bar:
- Left: small text "Changes restart the stack if it is running".
- Right: secondary "Reset", primary "Save and restart".

Include a danger zone at bottom, collapsed: "Remove site" with muted destructive styling.

SCREEN 3 - CREATE SITE MODAL
- Centered modal, 560px wide, dark surface, 16px radius.
- Title "Create site".
- Fields: Name, Domain (auto-derived preview), PHP version, Database switch.
- Show destination path in monospace: C:\Users\...\DevLok\sites\demo.
- Footer: Cancel, primary "Create and start".
- Include validation state: name empty, domain conflict.

SCREEN 4 - IMPORT LARAGON WIZARD
- Stepper: 1 Source, 2 Review, 3 Import.
- Step 1: path input "C:\laragon", detection summary, "Scan" button.
- Step 2: results table with checkbox per site, name, domain, root, source config, status.
- Dry-run notice: "No files will be changed until you confirm."
- Step 3: progress list with per-site result and a summary.
- Primary action "Import selected".

INTERACTION AND STATES
- Show hover, focus, disabled, loading, empty, error, and partial states for key components.
- Running badge uses dot plus label, never color alone.
- All controls keyboard focusable with a visible 2px blue focus ring.
- Status colors must remain readable on dark background.
- No fake charts or decorative analytics. This is a tool, not a dashboard template.
- Use realistic data, no lorem ipsum.

DELIVERABLES
- High-fidelity desktop screens at 1440x1024.
- Consistent spacing, typography, icons, and status language.
- A small component sheet: buttons, inputs, selects, switches, pills, tabs, table row, log line.
- Design should feel like a serious local dev tool a developer would use all day.
```

---

## Prompt Tray

```text
Design a compact system tray flyout for a Windows developer tool called "DevLok". This is not a full app screen. It is a 360 x 560 px popover that opens from the notification area.

STYLE
- Same design system as the DevLok desktop app.
- Background #0E1014, surface #15181F, border #272C38, text #E8EDF7.
- Primary blue #5B8DEF, success #2ED3A0, warning #F0B45A, danger #F06A7A.
- Inter or Geist, 13-14px base. Monospace for domains, ports, versions.
- Compact, dense, calm. No gradient, no glass, no emoji, no oversized cards.
- 12-14px radius, 1px borders, minimal shadow.

FLYOUT HEADER
- Left: small DevLok mark and wordmark.
- Right: global status pill "Running" with green dot, and a settings gear icon.
- Below header: active project "coba" and domain "coba.test" in monospace.
- URL row: "https://coba.test" with copy and open icons.

SERVICE LIST
- Section label "Services" with a tiny "All running" success note.
- Rows, 44px tall:
  1. MySQL - version chip "8.4.10" - port 3306 - green dot - power toggle on - chevron.
  2. Nginx - "1.30.6" - ports 80/443 - green dot - toggle on - chevron.
  3. PHP - "8.4.26" - port 19000 - green dot - toggle on - chevron.
- Each row opens a version submenu. Show a flyout state where the MySQL row is expanded with a version list:
  - 9.4.0
  - 8.4.10 (checked)
  - Each option has a check mark on the selected version.
- Include an alternate state: MySQL off, row muted, phpMyAdmin row hidden, warning line "Database disabled".
- Include partial state: MySQL failed, red dot, short error "port 3306 in use", action "Retry".

QUICK ACTIONS
- 2x2 grid of ghost buttons: Open site, phpMyAdmin, Folder, Terminal.
- Primary full-width button "Stop stack" in danger outline when running; "Start stack" in primary blue when stopped.

FOOTER
- Left: "DevLok 0.1.0".
- Right: "Quit" text button and a subtle "Open app" link.

TRAY ICON STATES
- Show two tray icon variants at 16px and 32px:
  1. Running: green circular mark with a white D, crisp at small sizes.
  2. Stopped: gray circular mark with a white D.
- Add a tooltip mock: "DevLok - coba.test running".
- Icon must be readable on light and dark Windows taskbars.

CONTEXT MENU VARIANT
- Also design a classic right-click context menu using the same tokens:
  - Show DevLok
  - Open site
  - phpMyAdmin
  - Nginx > 1.30.6 (checked)
  - PHP > 8.4.26 (checked)
  - MySQL > 9.4.0 / 8.4.10 (checked)
  - Separator
  - Stop stack
  - Quit
- Keep it compact, native-feeling, not oversized.

DELIVERABLE
- One high-fidelity 360x560 flyout.
- Expanded version picker state.
- Partial/error state.
- 16px and 32px tray icon states and tooltip.
- Design must feel native to Windows 11 while still belonging to DevLok.
```

---

## Prompt Tambahan (opsional)

Tempel jika ingin Stitch membuat lebih banyak layar:

```text
Add these screens using the same DevLok design system:
- Logs explorer: full-screen, filter by service, level, search, follow mode.
- Runtime manager: table of installed runtimes (Nginx, PHP, MySQL, Node, Composer, Adminer, Notepad++), versions, disk usage, remove action.
- Doctor: environment checks with ok/warn/fail rows and suggested fixes.
- Empty state: no sites yet, with "Create site" and "Import Laragon" actions.
- Error state: port 80 occupied, showing owner process and fallback port choice.
```

---

## Catatan untuk Stitch

- Jangan minta Stitch mendesain tray native sebenarnya. Stitch menghasilkan UI web. Frame tray di atas adalah mockup flyout; implementasi native tetap di Windows.
- Jika hasil terlalu ramai, tambahkan: "Reduce visual density by 20%. Remove decorative cards. Keep only functional grouping."
- Jika hasil terlihat seperti dashboard SaaS generik, tambahkan: "Make it feel like a desktop IDE utility, not an analytics dashboard. Use tables, monospace values, and inline controls."
- Jika versi select terlihat seperti form biasa, tambahkan: "Style version pickers as compact chips with a chevron, not default browser selects."
```
