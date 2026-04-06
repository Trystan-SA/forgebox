# ForgeBox Design System

This document defines the visual language for the ForgeBox web dashboard. All frontend components must follow these guidelines to maintain consistency.

## Tech Stack

- **Framework:** SvelteKit 2 + Svelte 5 (runes: `$state`, `$derived`, `$effect`, `$props`)
- **Styling:** SCSS with BEM-like naming (`block__element` via `&__` nesting)
- **Fonts:** Inter (sans), JetBrains Mono (mono)
- **Adapter:** Static (SPA mode with `fallback: 'index.html'`)
- **Platform:** Web + Tauri v2 desktop app

## SCSS Architecture

Styles live in `src/lib/styles/`:

| File | Purpose |
|------|---------|
| `_variables.scss` | Design tokens (colors, spacing, typography, shadows, radii, breakpoints) |
| `_mixins.scss` | Reusable style patterns (`@include card`, `@include input-base`, `@include btn`, etc.) |
| `_reset.scss` | CSS reset / normalize |
| `global.scss` | Imports all partials, defines base element styles and utility classes |

SvelteKit auto-injects `_variables.scss` and `_mixins.scss` into all `<style lang="scss">` blocks — use tokens and mixins directly, no `@use` needed in components.

## Color Palette

All colors use a numeric scale (50-900). Use semantic names, not hex values.

| Role | Token Prefix | Primary Use |
|------|-------------|-------------|
| **Primary** | `$primary-*` | Indigo. Buttons, links, focus rings, active states |
| **Neutral** | `$neutral-*` | Gray scale. Text, borders, backgrounds |
| **Success** | `$success-*` | Green. Positive states, completed tasks |
| **Warning** | `$warning-*` | Amber. Caution states, pending actions |
| **Error** | `$error-*` | Red. Errors, destructive actions, failed tasks |
| **Info** | `$info-*` | Blue. Informational messages |

### Key color assignments

- Page background: `$neutral-50`
- Card background: `$neutral-0` (white)
- Primary text: `$neutral-800` or `$neutral-900`
- Secondary text: `$neutral-500`
- Label text: `$neutral-700`
- Borders: `$neutral-200` (cards) or `$neutral-300` (inputs)
- Primary button: `$primary-600` bg, `$neutral-0` text, hover `$primary-700`
- Error messages: `$error-700` text, `$error-50` bg, `$error-100` border

## Typography

| Token | Size | Use |
|-------|------|-----|
| `$text-xs` | 0.75rem | Badges, fine print |
| `$text-sm` | 0.875rem | Form labels, body copy, buttons |
| `$text-base` | 1rem | Default body |
| `$text-lg` | 1.125rem | Section headings (h2) |
| `$text-2xl` | 1.5rem | Page titles (h1) |
| `$text-3xl` | 1.875rem | Hero / landing text |

Weights: `$font-normal` (400), `$font-medium` (500), `$font-semibold` (600), `$font-bold` (700)

## Spacing

4px base grid: `$space-1` (4px) through `$space-16` (64px).

Common patterns:
- Form gap between fields: `$space-4`
- Card padding: `$space-8`
- Section margin: `$space-6` to `$space-8`
- Tight inline spacing: `$space-1` to `$space-2`

## Components

### Cards
```scss
// Use the mixin — white bg, neutral-200 border, radius-xl, shadow-sm
@include card;
padding: $space-8;
```

### Inputs
```scss
// Use the mixin — full width, sm text, neutral-300 border, primary-500 focus ring
@include input-base;
```

### Buttons
Four global classes defined in `global.scss`:
- `.btn-primary` — indigo bg, white text (primary actions)
- `.btn-secondary` — white bg, neutral border (secondary actions)
- `.btn-danger` — red bg, white text (destructive actions)
- `.btn-ghost` — transparent bg, neutral text (tertiary actions)

All buttons use `$text-sm`, `$font-medium`, `$radius-lg`, and disable with `opacity: 0.5`.

### Badges
```scss
@include badge;
// Small rounded pill with xs text, used for status indicators
```

### Error/Alert Blocks
```scss
padding: $space-3;
font-size: $text-sm;
color: $error-700;
background: $error-50;
border: 1px solid $error-100;
border-radius: $radius-lg;
```

## Layout Conventions

- **Sidebar:** `$sidebar-width` (16rem) fixed left
- **Topbar:** `$topbar-height` (3.5rem) fixed top
- **Content:** Scrollable main area, padded with `$space-6` to `$space-8`
- Responsive breakpoints: `$bp-sm` (640), `$bp-md` (768), `$bp-lg` (1024), `$bp-xl` (1280)
- Use mixins `@include sm`, `@include md`, etc. for media queries

## Page Pattern — Full-Page Forms (Login, Setup)

Full-page centered forms follow this pattern:
- Outer wrapper: `@include flex-center; min-height: 100vh; background: $neutral-50`
- Card: `@include card; max-width: 400px; padding: $space-8`
- Header: Centered SVG icon + h1 + subtitle paragraph
- Form: Flex column with `gap: $space-4`
- Labels: `$text-sm`, `$font-medium`, `$neutral-700`
- Submit button: `.btn-primary` full width with `margin-top: $space-2`

## Naming Convention

Use BEM-like scoped SCSS in `<style lang="scss">`:
```scss
.page-name {
    // block
    &__element {
        // element
    }
}
```

The `onwarn` config in `svelte.config.js` suppresses false `css_unused_selector` warnings from `&__` patterns.

**Important:** Do not use `//` single-line comments inside `<style lang="scss">` blocks. Svelte's CSS parser runs before the SCSS preprocessor and will fail on them. Use `/* */` comments instead, or omit comments entirely.

## Accessibility

- All interactive elements must have focus-visible styles (buttons get `outline: 2px solid $primary-500`)
- Form inputs must have associated `<label>` elements
- Disabled states use `opacity: 0.5` + `pointer-events: none` or `cursor: not-allowed`
- Color alone never conveys meaning — pair with text or icons
