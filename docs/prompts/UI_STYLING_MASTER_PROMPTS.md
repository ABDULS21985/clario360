# Clario360 UI Styling Master Prompts

> **Goal:** Adopt the premium UI/UX styling from ITD-OPMS into the Clario360 frontend without modifying any business logic, data flows, API calls, state management, or routing.
>
> **Source Reference:** `/Users/mac/codes/itd-opms/itd-opms-portal`
> **Target:** `/Users/mac/clario360/frontend`

---

## PROMPT S-01: Global Design Tokens & Surface System

**Objective:** Replace the current minimal CSS custom properties in `globals.css` with a comprehensive 5-level surface system, premium shadows, glassmorphism tokens, fluid spacing, and refined color palette — while keeping the existing Clario360 brand colors (#1B5E20 green, #C6A962 gold, #0D4B4F teal).

**What to change:**
1. In `src/app/globals.css`, expand the `:root` CSS variables to include:
   - **5-level surface system:** `--surface-0: #FFFFFF` (cards/modals), `--surface-1: #F9FAFB` (page bg), `--surface-2: #F3F4F6` (table headers), `--surface-3: #E5E7EB` (borders), `--surface-4: #D1D5DB` (heavy dividers)
   - **Premium shadows:** `--shadow-sm`, `--shadow-md`, `--shadow-lg` plus `--shadow-premium: 0 25px 50px -12px rgba(27, 94, 32, 0.25)` and `--shadow-glow: 0 0 40px rgba(27, 94, 32, 0.3)` using Clario green
   - **Glassmorphism tokens:** `--glass-bg: rgba(255,255,255,0.7)`, `--glass-border: rgba(255,255,255,0.2)`, `--glass-shadow: 0 8px 32px rgba(0,0,0,0.1)`, `--glass-blur: 12px`
   - **Fluid spacing:** `--space-xs` through `--space-xl` using `clamp()` for responsive sizing
   - **Animation durations:** `--duration-fast: 150ms`, `--duration-normal: 300ms`, `--ease-out-expo: cubic-bezier(0.16, 1, 0.3, 1)`
   - **Additional accent tokens:** `--primary-light: #2E7D32`, `--gold: #C6A962`, `--gold-dark: #A68B42`, `--gold-light: #D4BE8A`
   - **Badge color tokens:** Complete set for blue, purple, red, amber, emerald (each with bg, text, dot variants)
2. In `tailwind.config.ts`, register the new surface levels, shadow tokens, and glass utilities as Tailwind extensions so they can be used as classes (e.g., `bg-surface-1`, `shadow-premium`)
3. Update the `.dark` section with proper dark-mode surface inversions (surface-0 → `#1E293B`, surface-1 → `#0F172A`, etc.) — keep the primary color as brand green in dark mode (fix the current bug where dark mode switches to blue)
4. Change `--background` to use `--surface-1` (#F9FAFB) instead of pure white, so pages have a subtle warm gray background with cards popping on white

**What NOT to change:** No component files, no logic, no routing — only `globals.css` and `tailwind.config.ts`.

---

## PROMPT S-02: Typography System & Font Upgrade

**Objective:** Upgrade from Inter to Aptos (with system-ui fallback chain) and implement a refined typographic scale with tighter letter-spacing on headings and improved font rendering.

**What to change:**
1. In `src/app/layout.tsx`:
   - Remove the `Inter` import from `next/font/google`
   - Add Aptos as local font files OR use a system-ui stack: `font-family: 'Aptos', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif` applied via inline style on `<body>`
   - Add CSS class `antialiased` to body for font smoothing
2. In `globals.css`, add typographic refinements:
   - `body { letter-spacing: -0.011em; }` for tighter body text
   - Headings (`h1-h6`): `letter-spacing: -0.02em; -webkit-font-smoothing: subpixel-antialiased;` for a glossy/premium feel
   - `.tabular-nums { font-variant-numeric: tabular-nums; }` utility for KPI/numeric values
3. In `tailwind.config.ts`, update the `fontFamily` to use the new Aptos stack as the sans-serif default
4. Ensure all existing `font-bold`, `text-2xl`, etc. classes continue to work — only the underlying font changes

**What NOT to change:** No component file modifications — font change cascades automatically through existing Tailwind classes.

---

## PROMPT S-03: Border Radius & Shape Language Upgrade

**Objective:** Shift the visual shape language from conservative `rounded-md`/`rounded-lg` to a more premium, softer aesthetic with `rounded-xl` as the default and `rounded-2xl`/`rounded-3xl` for cards and modals.

**What to change:**
1. In `tailwind.config.ts`, update the `--radius` CSS variable from `0.5rem` to `0.75rem` (this affects `rounded-lg`, `rounded-md`, `rounded-sm` computed values)
2. In `src/components/ui/button.tsx`: Change `rounded-md` to `rounded-xl` in the base variant class
3. In `src/components/ui/input.tsx`: Change `rounded-md` to `rounded-xl`
4. In `src/components/ui/card.tsx`: Change `rounded-lg` to `rounded-2xl` on the Card component
5. In `src/components/ui/dialog.tsx`: Change `sm:rounded-lg` to `rounded-2xl` on DialogContent
6. In `src/components/ui/badge.tsx`: Keep `rounded-full` (already correct)
7. In `src/components/shared/data-table/` table wrapper: Change `rounded-lg` to `rounded-xl`
8. In `src/components/layout/command-palette.tsx`: Change `rounded-xl` to `rounded-2xl`
9. In `src/components/ui/select.tsx`, `textarea.tsx`, and any dropdown trigger: Update to `rounded-xl`

**What NOT to change:** No logic changes. Badge `rounded-full` stays. Avatar circles stay. Only shape rounding values change.

---

## PROMPT S-04: Premium Shadow & Hover Effect System

**Objective:** Replace flat/minimal shadows with a layered shadow system featuring brand-colored glows, and add interactive hover-lift effects to cards and clickable elements.

**What to change:**
1. In `globals.css`, add utility classes:
   ```css
   .card-interactive {
     transition: transform var(--duration-normal) var(--ease-out-expo),
                 box-shadow var(--duration-normal) var(--ease-out-expo);
   }
   .card-interactive:hover {
     transform: translateY(-4px);
     box-shadow: var(--shadow-lg);
   }
   .hover-lift {
     transition: transform 200ms ease, box-shadow 200ms ease;
   }
   .hover-lift:hover {
     transform: translateY(-2px);
     box-shadow: 0 8px 25px rgba(0,0,0,0.1);
   }
   ```
2. In `src/components/ui/card.tsx`: Add `shadow-sm` as default (already present), and add a new `interactive` variant prop that applies the `card-interactive` class
3. In `src/components/dashboard/kpi-card.tsx`: Replace `hover:shadow-md` with the `card-interactive` class for premium hover lift
4. In `src/components/ui/button.tsx` (default variant only): Add `hover:shadow-md active:translate-y-0` for tactile button feedback
5. In `src/components/shared/status-badge.tsx` and `src/components/ui/badge.tsx`: No changes (badges should not lift)
6. In any card-based list item components: Add `hover-lift` class for subtle interactive feedback

**What NOT to change:** No click handlers, no state changes, no event logic.

---

## PROMPT S-05: Dark Sidebar with Gold Accents

**Objective:** Transform the sidebar from a light-themed panel to a dark-themed sidebar with gold shimmer section headers, matching the premium ITD-OPMS sidebar aesthetic, while keeping all navigation logic, collapsible behavior, and routing intact.

**What to change:**
1. In `src/components/layout/sidebar.tsx`:
   - Change the `<aside>` background from `bg-card` to a dark gradient: `bg-gradient-to-b from-[#031A0B] to-[#0A2E12]` (deep dark green, derived from Clario brand)
   - Text colors: nav items → `text-gray-300`, muted text → `text-gray-500`, active item → `text-white`
   - Active nav item: Replace `bg-primary/10 text-primary border-l-2 border-primary` with `bg-white/10 text-white border-l-3 border-gradient` (gradient left bar from green to emerald)
   - Hover state: `hover:bg-white/5 hover:text-white`
   - Border right: change `border-r` to `border-r border-white/10`
   - Logo text "Clario 360": change to `text-white` with the gold color applied to "360": `text-[#C6A962]`
2. In `src/components/layout/sidebar.tsx` SidebarSection headers:
   - Style section labels with gold shimmer gradient text: `background: linear-gradient(135deg, #C6A962, #A68B42, #D4BE8A, #C6A962); -webkit-background-clip: text; -webkit-text-fill-color: transparent;`
   - Add `text-[11px] font-bold uppercase tracking-[0.15em]`
3. In `globals.css`, add the sidebar gold shimmer animation:
   ```css
   .sidebar-gold-text {
     background: linear-gradient(135deg, #C6A962 0%, #A68B42 40%, #D4BE8A 60%, #C6A962 100%);
     background-size: 200% auto;
     -webkit-background-clip: text;
     -webkit-text-fill-color: transparent;
     animation: sidebar-gold-shimmer 3s ease-in-out infinite;
   }
   @keyframes sidebar-gold-shimmer {
     0%, 100% { background-position: 0% center; }
     50% { background-position: 100% center; }
   }
   ```
4. Sidebar collapse toggle and user footer: dark-theme compatible colors (`text-gray-400`, `hover:bg-white/10`)
5. Sidebar scrollbar: add custom scrollbar styling (`4px wide, rgba(255,255,255,0.1)` thumb)
6. `MobileSidebar` sheet: update to match the same dark theme

**What NOT to change:** All navigation items, routes, collapse logic, Zustand store, permission checks, badge counts — everything functional stays identical.

---

## PROMPT S-06: Header Refinement & Search Pill

**Objective:** Upgrade the header from a basic toolbar to a polished sticky bar with styled breadcrumbs (chevron-separated), a search pill trigger (rounded, pill-shaped), and refined action icons.

**What to change:**
1. In `src/components/layout/header.tsx`:
   - Keep `sticky top-0 z-30 h-16 border-b` but enhance background: `bg-[var(--surface-0)]/80 backdrop-blur-md` for subtle glass effect on scroll
   - Search button: restyle from icon-only to a pill shape: `flex items-center gap-2 rounded-xl bg-[var(--surface-1)] border border-[var(--border)] px-3 py-1.5 text-sm text-muted-foreground hover:border-primary/30` with `Cmd+K` hint text
   - Right-side icons (notification bell, user menu): add `rounded-xl` and `hover:bg-accent` consistent styling
2. In `src/components/layout/breadcrumbs.tsx`:
   - Ensure chevron separators use `ChevronRight` icon (14px, text-muted-foreground)
   - Active/current breadcrumb: `font-medium text-foreground`
   - Parent links: `text-muted-foreground hover:text-foreground transition-colors`
3. Add a subtle bottom shadow on scroll (can be done with an intersection observer or a simple `shadow-sm` that's always present)

**What NOT to change:** Breadcrumb logic, notification data, user menu actions, Cmd+K trigger — all functional behavior stays.

---

## PROMPT S-07: Glassmorphism Card System

**Objective:** Introduce glassmorphism-styled cards as an alternative card variant for dashboard widgets, KPI cards, and feature panels — semi-transparent backgrounds with backdrop blur and subtle borders.

**What to change:**
1. In `globals.css`, add glassmorphism utility classes:
   ```css
   .glass-card {
     background: var(--glass-bg);
     backdrop-filter: blur(var(--glass-blur));
     -webkit-backdrop-filter: blur(var(--glass-blur));
     border: 1px solid var(--glass-border);
     box-shadow: var(--glass-shadow);
   }
   .glass-card-dark {
     background: rgba(15, 23, 42, 0.6);
     backdrop-filter: blur(12px);
     border: 1px solid rgba(255, 255, 255, 0.1);
   }
   ```
2. In `src/components/ui/card.tsx`, add a `glass` variant:
   - When `variant="glass"`, apply the `glass-card` class instead of `bg-card shadow-sm`
   - Keep the existing `default` variant unchanged
3. In `src/components/dashboard/kpi-card.tsx`: Switch to the glass variant for dashboard KPI cards
4. In `src/components/shared/kpi-card.tsx`: Add optional `glass` prop to enable glassmorphism styling
5. Ensure dark mode compatibility: when `.dark`, use `glass-card-dark` colors

**What NOT to change:** Card content structure, data binding, click handlers — only the visual surface changes.

---

## PROMPT S-08: Premium Button Styles & Gradient CTAs

**Objective:** Upgrade buttons with gradient backgrounds for primary CTAs, hover shadow-lift effects, shimmer animation on key action buttons, and tactile active-press states.

**What to change:**
1. In `src/components/ui/button.tsx`, add new variants:
   - `gradient`: `bg-gradient-to-r from-[#1B5E20] to-[#0A3D12] text-white shadow-lg hover:shadow-xl hover:-translate-y-0.5 active:translate-y-0 transition-all`
   - `gradient-gold`: `bg-gradient-to-r from-[#C6A962] to-[#A68B42] text-white shadow-lg hover:shadow-xl hover:-translate-y-0.5 active:translate-y-0`
   - Keep all existing variants (`default`, `destructive`, `outline`, `secondary`, `ghost`, `link`) unchanged
2. In `globals.css`, add the button shimmer animation:
   ```css
   .btn-shimmer::after {
     content: '';
     position: absolute;
     top: 0; left: -100%; width: 100%; height: 100%;
     background: linear-gradient(90deg, transparent, rgba(255,255,255,0.2), transparent);
     animation: btn-shimmer 3s infinite;
   }
   @keyframes btn-shimmer {
     0% { left: -100%; }
     100% { left: 100%; }
   }
   .btn-shimmer { position: relative; overflow: hidden; }
   ```
3. Add `btn-shimmer` class to the `gradient` variant for premium shimmer effect
4. Update the default variant to add `active:scale-[0.98]` for tactile press feel

**What NOT to change:** Button onClick handlers, form submissions, disabled states logic, loading states.

---

## PROMPT S-09: Enhanced Form Input Styling

**Objective:** Upgrade form inputs with premium focus effects (glowing ring + border color change), animated error shake, success checkmark animation, and refined hover states.

**What to change:**
1. In `src/components/ui/input.tsx`:
   - Add hover state: `hover:border-primary/30`
   - Enhance focus: `focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:border-primary` (replace generic ring with brand-colored ring)
   - Add premium focus glow via box-shadow: `focus-within:shadow-[0_0_0_3px_rgba(27,94,32,0.08),0_0_20px_rgba(27,94,32,0.06)]`
2. In `src/components/ui/textarea.tsx`: Apply the same hover/focus enhancements
3. In `src/components/ui/select.tsx` (trigger): Apply matching focus/hover styling
4. In `globals.css`, add the error shake animation:
   ```css
   @keyframes shake-error {
     0%, 100% { transform: translateX(0); }
     15% { transform: translateX(-6px); }
     30% { transform: translateX(5px); }
     45% { transform: translateX(-4px); }
     60% { transform: translateX(3px); }
     75% { transform: translateX(-2px); }
   }
   .animate-shake-error { animation: shake-error 0.4s ease-out; }
   ```
5. In form field components (`src/components/forms/form-field.tsx`):
   - When field has error: apply `border-destructive ring-2 ring-destructive/10 animate-shake-error`
   - When field is valid after edit: apply `border-green-500` with a brief checkmark icon transition
6. Labels: ensure `text-sm font-medium text-foreground mb-1.5` with red asterisk for required fields

**What NOT to change:** Validation logic, react-hook-form integration, Zod schemas, onSubmit handlers.

---

## PROMPT S-10: DataTable Premium Styling

**Objective:** Upgrade the DataTable with premium header styling, glassmorphism bulk-action toolbar, keyboard navigation visual feedback, and refined pagination.

**What to change:**
1. In `src/components/shared/data-table/`:
   - Table container: `rounded-xl border border-[var(--border)] bg-[var(--surface-0)] shadow-sm overflow-hidden`
   - Header row: `bg-[var(--surface-1)]` background, `text-xs font-semibold uppercase tracking-wider text-[var(--neutral-gray)]`
   - Body rows: `hover:bg-[var(--surface-1)]` with smooth transition, selected: `bg-primary/5`
   - Active sort column indicator: colored arrow in `text-primary`
2. Bulk action toolbar:
   - Glassmorphism style: `bg-primary/10 backdrop-blur-sm rounded-xl border border-primary/20 px-4 py-2`
   - Slide-in animation from top using CSS transform
3. Pagination redesign:
   - Active page: `bg-primary text-white rounded-lg h-8 w-8`
   - Other pages: `hover:bg-accent rounded-lg h-8 w-8`
   - First/Last page buttons alongside Prev/Next
   - Per-page size selector: `rounded-lg border bg-surface-0 text-xs`
4. Empty state: centered with `rounded-2xl bg-surface-2` icon container and descriptive text
5. Row hover: subtle left-border accent that appears on hover (`border-l-2 border-transparent hover:border-l-2 hover:border-primary/50`)

**What NOT to change:** Column definitions, sorting logic, filtering, data fetching, tanstack/react-table integration, bulk action handlers.

---

## PROMPT S-11: Bento Dashboard KPI Grid

**Objective:** Transform the dashboard KPI card grid from a uniform 4-column grid into a dynamic bento layout with hero (2x2), wide (2x1), and compact (1x1) card sizes, with staggered entry animations.

**What to change:**
1. In `src/components/dashboard/kpi-grid.tsx` (or the dashboard page):
   - Replace `grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4` with a bento grid: `grid grid-cols-2 sm:grid-cols-4 gap-4 auto-rows-min`
   - First/hero KPI card: `col-span-2 row-span-2` with larger value text (`text-4xl font-bold tabular-nums`)
   - Important KPI cards: `col-span-2` (wide format)
   - Standard KPI cards: `col-span-1` (compact format)
2. In `src/components/dashboard/kpi-card.tsx`:
   - Add a `size` prop: `'hero' | 'wide' | 'compact'` with default `'compact'`
   - Hero: larger padding (`p-8`), bigger icon (`h-14 w-14`), value in `text-4xl`, includes optional sparkline/trend mini-chart
   - Wide: standard padding (`p-6`), value in `text-3xl`, two-column internal layout (value left, chart right)
   - Compact: current size (`p-4-6`), value in `text-2xl`
3. Add staggered entry animations using Framer Motion (if available) or CSS:
   ```css
   .bento-card { animation: bento-enter 0.4s var(--ease-out-expo) both; }
   .bento-card:nth-child(1) { animation-delay: 0ms; }
   .bento-card:nth-child(2) { animation-delay: 80ms; }
   /* ... up to 8 cards */
   @keyframes bento-enter {
     from { opacity: 0; transform: translateY(12px) scale(0.98); }
     to { opacity: 1; transform: translateY(0) scale(1); }
   }
   ```
4. Add `@media (prefers-reduced-motion: reduce)` to disable animations

**What NOT to change:** KPI data sources, API calls, metric calculations, navigation on click.

---

## PROMPT S-12: Status Badge Animations & Severity Enhancements

**Objective:** Add visual polish to status badges and severity indicators with animated colored dots, pulse effects for active/warning states, and improved dark mode contrast.

**What to change:**
1. In `src/components/shared/status-badge.tsx`:
   - For `dot` variant: add a pulsing animation on warning/critical statuses
   ```css
   .badge-dot-pulse {
     animation: dot-pulse 2s ease-in-out infinite;
   }
   @keyframes dot-pulse {
     0%, 100% { transform: scale(1); opacity: 1; }
     50% { transform: scale(1.4); opacity: 0.7; }
   }
   ```
   - Critical dot: add a red glow `box-shadow: 0 0 8px rgba(239, 68, 68, 0.6)`
   - Warning dot: add pulse animation class
   - Success dot: solid, no animation (stable state)
2. In `src/components/shared/severity-indicator.tsx`:
   - Maintain all existing severity levels and color mappings
   - Add left-edge color stripe option: `border-l-4` with severity-appropriate color for list contexts
   - Ensure dark mode colors maintain WCAG contrast ratios (the existing dark mode colors are good, just verify)
3. In `src/components/ui/badge.tsx`:
   - Add optional `pulse` prop that enables the pulse animation
   - Add optional `dot` prop that prepends a small colored circle before the text

**What NOT to change:** Badge variant logic, severity level mapping, conditional rendering logic.

---

## PROMPT S-13: Dialog & Modal Premium Styling

**Objective:** Upgrade dialogs and modals with backdrop blur, larger border radius, icon header boxes, and refined action button layout.

**What to change:**
1. In `src/components/ui/dialog.tsx`:
   - Overlay: change `bg-black/80` to `bg-black/50 backdrop-blur-sm` for a softer, more premium overlay
   - Content: change `sm:rounded-lg` to `rounded-2xl`, add `shadow-xl`
   - Increase default padding to `p-6` if not already
2. In `src/components/shared/confirm-dialog.tsx`:
   - Add an icon header box: `rounded-xl w-12 h-12 flex items-center justify-center` with variant-colored background
     - Danger: `bg-red-100 text-red-600`
     - Warning: `bg-amber-100 text-amber-600`
     - Default: `bg-primary/10 text-primary`
   - Button layout: Cancel button as `outline` variant (left), Confirm button as filled/gradient (right), with `gap-3`
   - Auto-focus the confirm button on open for accessibility
3. In `src/components/shared/detail-panel.tsx` (Sheet):
   - Ensure the Sheet overlay uses `backdrop-blur-sm`
   - Sheet content: `rounded-l-2xl` for slide-in panels
4. In `src/components/layout/command-palette.tsx`:
   - Overlay: `backdrop-blur-sm` instead of just opacity
   - Panel: already `rounded-xl` — update to `rounded-2xl shadow-2xl`

**What NOT to change:** Dialog open/close logic, form submissions in modals, keyboard shortcuts, focus trap logic.

---

## PROMPT S-14: Page Transition Animations

**Objective:** Add smooth page entry animations with opacity/translate transitions on route changes, and staggered section reveals within pages.

**What to change:**
1. Install `framer-motion` if not already present (check package.json first — it may already be a dependency)
2. Create `src/components/common/page-transition.tsx`:
   ```tsx
   // Wrapper component for page content with entry animation
   // opacity: 0 → 1, y: 8 → 0, duration: 200ms, ease: [0.16, 1, 0.3, 1]
   ```
3. In `src/app/(dashboard)/layout.tsx`, wrap `{children}` with the PageTransition component
4. Create `src/components/common/stagger-container.tsx`:
   ```tsx
   // Container that staggers children's entry animations
   // Each child: opacity: 0 → 1, y: 12 → 0, stagger delay: 50ms per child
   ```
5. In `globals.css`, add CSS-only fallback animations:
   ```css
   .animate-page-enter {
     animation: page-enter 0.2s cubic-bezier(0.16, 1, 0.3, 1) both;
   }
   @keyframes page-enter {
     from { opacity: 0; transform: translateY(8px); }
     to { opacity: 1; transform: translateY(0); }
   }
   .animate-stagger-in {
     animation: stagger-in 0.3s var(--ease-out-expo) both;
   }
   @keyframes stagger-in {
     from { opacity: 0; transform: translateY(12px); }
     to { opacity: 1; transform: translateY(0); }
   }
   ```
6. Add `@media (prefers-reduced-motion: reduce)` to suppress all animations

**What NOT to change:** Routing logic, page component structure, data fetching, suspense boundaries.

---

## PROMPT S-15: Loading States & Skeleton Enhancements

**Objective:** Upgrade loading skeletons with improved shimmer effects, floating animations on empty states, and component-specific skeleton variants.

**What to change:**
1. In `globals.css`, enhance the skeleton shimmer:
   ```css
   .skeleton-shimmer {
     background: linear-gradient(
       90deg,
       hsl(var(--muted)) 0%,
       hsl(var(--muted) / 0.4) 50%,
       hsl(var(--muted)) 100%
     );
     background-size: 200% 100%;
     animation: shimmer 1.5s ease-in-out infinite;
   }
   @keyframes shimmer {
     0% { background-position: 200% 0; }
     100% { background-position: -200% 0; }
   }
   ```
2. In `src/components/ui/skeleton.tsx`: Replace `animate-pulse` with `skeleton-shimmer` for a more premium loading effect
3. In `src/components/common/loading-skeleton.tsx`:
   - Ensure all variants (`card`, `table-row`, `list-item`, `text`, `avatar`, `chart`) use the new shimmer
   - Add `bento` variant for the bento grid layout (hero + wide + compact skeleton shapes)
4. In `src/components/common/empty-state.tsx`:
   - Add floating animation to the icon:
   ```css
   .float-empty {
     animation: float-empty 3s ease-in-out infinite;
   }
   @keyframes float-empty {
     0%, 100% { transform: translateY(0); }
     50% { transform: translateY(-8px); }
   }
   ```
   - Apply `float-empty` class to the empty state icon container
5. Dark mode: reduce shimmer opacity to `rgba(255,255,255,0.06)` base

**What NOT to change:** Loading state trigger logic, data fetching states, error boundaries.

---

## PROMPT S-16: Sidebar Icon Hover Micro-Animations

**Objective:** Add per-icon hover micro-animations to sidebar navigation items — subtle scale, rotate, bounce, or wave effects that activate when hovering over each nav item.

**What to change:**
1. In `globals.css`, add sidebar icon animation keyframes:
   ```css
   .sidebar-icon-pulse { }
   .sidebar-icon-pulse:hover svg { animation: sidebar-pulse 500ms ease; }
   @keyframes sidebar-pulse { 0%,100% { transform: scale(1); } 50% { transform: scale(1.15); } }

   .sidebar-icon-bounce:hover svg { animation: sidebar-bounce 400ms ease; }
   @keyframes sidebar-bounce { 0%,100% { transform: translateY(0); } 40% { transform: translateY(-3px); } }

   .sidebar-icon-rotate:hover svg { animation: sidebar-rotate 400ms ease; }
   @keyframes sidebar-rotate { 0% { transform: rotate(0); } 100% { transform: rotate(15deg); } }

   .sidebar-icon-wave:hover svg { animation: sidebar-wave 500ms ease; }
   @keyframes sidebar-wave { 0%,100% { transform: translateY(0); } 30% { transform: translateY(-2px); } 60% { transform: translateY(2px); } }
   ```
2. In `src/config/navigation.ts` (or sidebar nav item component):
   - Map each navigation section to a hover animation class:
     - Dashboard → `sidebar-icon-pulse`
     - Security/Cyber → `sidebar-icon-rotate`
     - Compliance → `sidebar-icon-bounce`
     - Tasks/Workflow → `sidebar-icon-wave`
     - Settings → `sidebar-icon-rotate`
   - Apply the animation class to each nav item's icon wrapper `<span>` or `<div>`
3. Add `@media (prefers-reduced-motion: reduce)` to disable all sidebar animations

**What NOT to change:** Navigation routing, active state logic, permission checks, badge counts.

---

## PROMPT S-17: Notification Dropdown & Toast Polish

**Objective:** Upgrade the notification dropdown and toast notifications with slide-in animations, severity-colored accents, and improved visual hierarchy.

**What to change:**
1. In `src/components/layout/notification-dropdown.tsx`:
   - Dropdown panel: add `rounded-2xl shadow-2xl border` with smooth `animate-dropdown` entry
   - Each notification item: add a left color stripe (`border-l-3`) based on notification type/severity
     - Error/Critical: `border-l-red-500`
     - Warning: `border-l-amber-500`
     - Info: `border-l-blue-500`
     - Success: `border-l-green-500`
   - Unread items: subtle `bg-primary/5` background tint
   - Hover: `hover:bg-surface-1` with smooth transition
   - Timestamp: `text-xs text-muted-foreground` with relative time
2. In `globals.css`, add dropdown animation:
   ```css
   .animate-dropdown {
     animation: dropdown 150ms var(--ease-out-expo);
   }
   @keyframes dropdown {
     from { opacity: 0; transform: translateY(4px) scale(0.98); }
     to { opacity: 1; transform: translateY(0) scale(1); }
   }
   ```
3. In toast configuration (`src/lib/toast.ts` or Sonner config):
   - Ensure toasts use `rounded-xl` border radius
   - Success toasts: green left accent stripe
   - Error toasts: red left accent stripe
   - Position: bottom-right with proper z-index

**What NOT to change:** Notification data fetching, mark-as-read logic, WebSocket subscription, toast trigger logic.

---

## PROMPT S-18: Chart Component Visual Polish

**Objective:** Upgrade chart styling with brand-consistent colors, refined grid lines, polished tooltips, and consistent dark-mode support.

**What to change:**
1. In chart components (`src/components/shared/charts/`):
   - Default color palette: use Clario brand-derived colors:
     - Primary: `#1B5E20` (brand green)
     - Secondary: `#C6A962` (gold)
     - Tertiary: `#0D4B4F` (teal)
     - Additional: `#3B82F6` (blue), `#8B5CF6` (purple), `#F59E0B` (amber), `#EF4444` (red), `#06B6D4` (cyan)
   - Grid lines: `stroke="hsl(var(--border))"` with `strokeDasharray="3 3"` for subtle dashed lines
   - Axis text: `fontSize: 12, fill: "hsl(var(--muted-foreground))"`, hide axis lines and tick lines
   - Bar chart bars: `radius={[4, 4, 0, 0]}` (rounded top corners)
2. In ChartContainer (`src/components/shared/charts/chart-container.tsx` or similar wrapper):
   - Tooltip: `rounded-xl bg-popover border shadow-lg p-3` with proper dark mode colors
   - Legend: `iconType="circle" iconSize={8}` for consistent legend dots
3. In KPI cards with trend indicators:
   - Positive trend: `text-green-600` with `TrendingUp` icon
   - Negative trend: `text-red-500` with `TrendingDown` icon
   - Use `tabular-nums` class for numeric values in charts

**What NOT to change:** Data series, chart types, responsive sizing, data transformation logic.

---

## PROMPT S-19: Mobile Bottom Navigation & Sheet Animations

**Objective:** Add a mobile bottom tab bar for primary navigation sections (visible only on small screens) and upgrade the mobile sidebar sheet with spring animations and a handle indicator.

**What to change:**
1. Create `src/components/layout/mobile-bottom-nav.tsx`:
   - Fixed bottom bar, `h-14`, visible only on `lg:hidden`
   - 4 primary tabs + "More" button (opens existing MobileSidebar)
   - Tab items: icon (20px) + label (10px) stacked vertically
   - Active tab: `text-primary`, inactive: `text-muted-foreground`
   - Background: `bg-card/80 backdrop-blur-md border-t`
   - Safe area padding for notched devices: `pb-[env(safe-area-inset-bottom)]`
2. In `src/app/(dashboard)/layout.tsx`:
   - Add `<MobileBottomNav />` inside the layout (outside the main scroll area)
   - Add `pb-14 lg:pb-0` to the main content area to account for the bottom nav
3. In `src/components/layout/mobile-sidebar.tsx` (Sheet component):
   - Add a drag handle indicator at the top: `w-10 h-1 rounded-full bg-muted-foreground/30 mx-auto mt-2`
   - Sheet content: `rounded-t-2xl` for bottom-sheet feel (if it slides from bottom) or keep right-slide with `rounded-l-2xl`
4. Configure the 4 primary tabs based on existing navigation config (Dashboard, Cyber, Compliance, Tasks — or whatever the top-level sections are)

**What NOT to change:** Navigation config data, route definitions, sidebar content, authentication checks.

---

## PROMPT S-20: Reduced Motion, Print Styles & Theme Transition Polish

**Objective:** Add comprehensive `prefers-reduced-motion` support, print-friendly styles, and smooth theme-transition animations for the light/dark mode toggle.

**What to change:**
1. In `globals.css`, add comprehensive reduced motion support:
   ```css
   @media (prefers-reduced-motion: reduce) {
     *, *::before, *::after {
       animation-duration: 0.01ms !important;
       animation-iteration-count: 1 !important;
       transition-duration: 0.01ms !important;
       scroll-behavior: auto !important;
     }
     .card-interactive:hover,
     .hover-lift:hover {
       transform: none !important;
     }
   }
   ```
2. Add theme transition smoothing:
   ```css
   .theme-transitioning,
   .theme-transitioning *,
   .theme-transitioning *::before,
   .theme-transitioning *::after {
     transition: color 300ms ease, background-color 300ms ease,
                 border-color 300ms ease, box-shadow 300ms ease,
                 fill 300ms ease, stroke 300ms ease !important;
   }
   ```
   - In the theme toggle logic (wherever dark mode is toggled), add/remove the `theme-transitioning` class to `document.documentElement` for 400ms during the switch
3. Add print styles:
   ```css
   @media print {
     .no-print, nav, header, aside, [role="navigation"],
     .mobile-bottom-nav, .connection-banner { display: none !important; }
     body { background: white !important; color: black !important; }
     .glass-card { background: white !important; backdrop-filter: none !important; }
     * { box-shadow: none !important; }
     main { max-width: 100% !important; padding: 0 !important; }
   }
   ```
4. In the root layout or theme provider, add the theme flash prevention script:
   ```tsx
   // Inline script in <head> that reads localStorage theme before first paint
   // to prevent white flash when user prefers dark mode
   ```
5. Ensure all 19 previous prompt animations respect `prefers-reduced-motion`

**What NOT to change:** Theme toggle logic, localStorage keys, system preference detection, any JavaScript behavior.

---

## Implementation Order (Recommended)

| Phase | Prompts | Description |
|-------|---------|-------------|
| **1 — Foundation** | S-01, S-02, S-03 | Design tokens, typography, shape language |
| **2 — Core Components** | S-04, S-08, S-09 | Shadows, buttons, form inputs |
| **3 — Shell** | S-05, S-06, S-19 | Sidebar, header, mobile nav |
| **4 — Cards & Glass** | S-07, S-11, S-12 | Glassmorphism, bento grid, badges |
| **5 — Data & Content** | S-10, S-18 | Tables, charts |
| **6 — Interactions** | S-13, S-14, S-16, S-17 | Dialogs, transitions, micro-animations, notifications |
| **7 — Polish** | S-15, S-20 | Loading states, accessibility, print, theme transitions |

---

## Key Principles

1. **Style-only changes** — No modifications to state management, API calls, routing, data flows, or business logic
2. **Brand preservation** — Keep Clario360 brand colors (#1B5E20, #C6A962, #0D4B4F) while adopting the premium visual language
3. **Progressive enhancement** — Each prompt is self-contained and can be applied independently
4. **Accessibility first** — Every animation respects `prefers-reduced-motion`, every contrast ratio meets WCAG AA
5. **Dark mode aware** — All visual changes work in both light and dark themes
