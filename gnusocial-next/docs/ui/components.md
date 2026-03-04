# Design System

## Design Tokens
- Spacing scale: `4 / 8 / 12 / 16 / 24 / 32`
- Center content max width: `680px` to `760px` responsive
- Layout breakpoints:
  - desktop: 3-column
  - tablet: 2-column
  - mobile: 1-column
- Typography scale: `12 / 14 / 16 / 18 / 20 / 24 / 32` with consistent line-height per tier

## Density Modes
- Comfortable
  - Larger paddings
  - Larger media previews
- Default
  - Balanced spacing and media
- Compact
  - Tighter cards
  - Smaller avatars
  - Denser lists

## Core Components
- AppShell
- TopBar (sticky center header controls)
- SidebarNav (primary and secondary sections)
- CommandPalette (Ctrl+K)
- PostCard
- ThreadView (spine and nesting)
- Composer (thread composer, CW, visibility, language)
- MediaGrid (fixed aspect slots to prevent layout shift)
- FiltersBar (chips and dropdowns)
- Tabs
- Toasts
- Modal / Drawer
- EmptyState
- Skeletons
- VirtualList (optional, later phase)

## Accessibility Rules
- Visible focus rings on all interactive controls
- Full keyboard navigation support
- ARIA labels for icon-only actions
- Reduced motion toggle in user settings
