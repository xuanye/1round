# 一局一分 · UI Design Guidelines

> Scope: all pages, components, and future features of the WeChat Mini Program.
> This document governs visual design and interaction presentation. The PRD and domain rules govern feature scope, scoring rules, and permissions.
> Status: based on the generated high-fidelity homepage direction. The values below are implementation standards, not exact measurements sampled from the image.
> This document is written in English. Quoted Chinese UI labels remain the actual product copy; do not translate the interface as part of applying these guidelines.

## 1. Design Direction

A warm, clear, restrained everyday tool for the card table. Use a paper-white background, forest-green actions, and small terracotta score accents to create an approachable, orderly interface.

Establish quality through typography, alignment, spacing, and carefully proportioned components. Content and actions take priority; decoration provides only a small amount of brand identity.

- Give each page one primary visual focus, usually the main action or key result for the current task.
- Organize content with the page background, whitespace, and thin dividers. Use cards only for information groups that need a distinct boundary.
- Limit combinations of font size, weight, and color. Avoid making every heading, number, and button bold.
- For pages with established functionality, improve visual hierarchy without removing features or changing workflows for stylistic consistency.
- Reuse these guidelines and existing components on new pages instead of inventing a separate visual language.

## 2. Color Tokens

| Token                   | Value     | Usage                                          |
| ----------------------- | --------- | ---------------------------------------------- |
| `color-bg`              | `#F8F7F3` | Paper-white page background                    |
| `color-surface`         | `#FFFFFF` | Necessary independent surfaces and overlays    |
| `color-text`            | `#202925` | Headings, body text, and primary data          |
| `color-text-secondary`  | `#68716D` | Dates, participant counts, and descriptions    |
| `color-text-disabled`   | `#A3AAA6` | Disabled content; never essential instructions |
| `color-primary`         | `#175D50` | Primary actions, selected states, and branding |
| `color-primary-pressed` | `#124B40` | Pressed primary actions                        |
| `color-primary-subtle`  | `#EAF0EC` | Small selected-state backgrounds               |
| `color-border`          | `#DDE3DE` | List dividers and subtle borders               |
| `color-score-positive`  | `#B95740` | Positive scores; restrained terracotta         |
| `color-score-negative`  | `#526D65` | Negative scores; muted gray-green              |
| `color-danger`          | `#B33E38` | Destructive actions and errors                 |

Always show explicit `+` and `−` signs for positive and negative scores respectively; never rely on color alone. Use the secondary text color for zero. Positive-score color expresses a score, not a success state. Use the primary color with clear text or an icon for success feedback.

Use solid backgrounds. Do not add paper textures, noise, metallic effects, or button gradients. Body text must have at least 4.5:1 contrast against its background. Interactive boundaries and focus indicators must be clearly visible.

## 3. Typography and Sizing

Use system sans-serif fonts, prioritizing the system Chinese font. Chinese text, English text, buttons, and scores must share a consistent font stack. Tabular numerals are appropriate; do not introduce serif numerals.

All dimensions below are logical pixels, not pixels in an exported image. Convert them using the Mini Program's existing adaptation strategy. Do not copy dimensions directly from a high-resolution mockup.

| Level                   | Font size / weight | Usage                                   |
| ----------------------- | ------------------ | --------------------------------------- |
| Main page heading       | 26–28 / 600        | Homepage headline; at most one per page |
| Page or section heading | 18–20 / 600        | Navigation titles and section names     |
| Body / list title       | 16 / 400–500       | Names and body text                     |
| Button                  | 16 / 600           | Primary and secondary actions           |
| Supporting text         | 13–14 / 400        | Dates, counts, and hints                |
| Bottom navigation label | 11–12 / 500        | The three tabs                          |
| List score              | 24–28 / 600        | Historical results                      |
| Main score              | 32–40 / 600        | Score input and key settlement results  |

Use approximately 1.5 line height for body text and 1.25 for headings. Handle long names according to the actual container width; do not force them to fit by compressing letter spacing or shrinking text to an unreadable size. Right-align scores and prevent large values from overlapping names.

## 4. Layout, Spacing, and Shape

- Design for portrait mobile use with a 390px logical-width baseline. Check at least 320, 375, 390, and 430px widths.
- Default horizontal page padding is 20px. More spacious pages may use 24px; keep padding consistent within each page.
- Use the spacing scale `4 / 8 / 12 / 16 / 24 / 32`. Related elements should be closer together than separate sections.
- Primary buttons are 48–52px tall. Independent touch targets must be at least 44×44px.
- Standard control corner radii are 10–12px; standalone cards use 12–16px. Avoid pill shapes throughout the interface.
- List rows should generally be 72–84px tall, adapting to content. Fixed heights must not clip text.
- Allow natural content scrolling. Reserve space for fixed action areas, bottom navigation, and safe-area insets so the last row is not obscured.
- Do not enlarge cards or create excessive whitespace merely to fill the screen. Do not reduce touch targets to make the layout denser.
- Use no shadows by default. A subtle shadow is acceptable only for overlays that genuinely float above content.

### Shared spacing tokens

`apps/miniprogram/src/theme.wxss` owns spacing values. Pages, components, and inline WXML styles must reference these tokens instead of repeating numeric margin, padding, or gap values.

| Token | Default logical size | Typical use |
| --- | --- | --- |
| `--space-xs` | 4px | Text metadata and icon-label separation |
| `--space-sm` | 8px | Closely related controls and small groups |
| `--space-md` | 12px | Labels, descriptions, and control padding |
| `--space-lg` | 16px | Content padding and related content groups |
| `--space-xl` | 24px | Separate content groups |
| `--space-2xl` | 32px | Major sections |

- `--spacing-density` defaults to `1`. It adjusts content margin, padding, and gap across the app without changing font sizes, line heights, minimum control heights, or touch-target sizes. Keep the default for structural token migrations; evaluate a density change visually before adopting it.
- `--page-inset` remains the default 20px horizontal page inset. Density does not change this token.
- Older pages use `rpx`, which scales with device width. The `--space-responsive-*` counterparts preserve this behavior (8/16/24/32/48/64rpx respectively); do not substitute a px token for an rpx token during a visual-preserving refactor.
- `--space-legacy-*` values preserve existing off-scale spacing during migration and follow the same density setting. They are compatibility values, not new approved spacing variants. New or redesigned UI must choose the six standard sizes; gradually retire compatibility values when those screens are visually reviewed.
- `--layout-reserve-*`, `--tab-height`, and `env(safe-area-inset-bottom)` reserve space for fixed controls or platform safe areas. They do not scale with content density. Positional icon offsets, illustration geometry, borders, and dimensions are not content spacing.
- Preserve `0`, `auto`, and dynamic safe-area calculations where appropriate. Use a negative token through `calc(0rpx - var(...))` when an existing layout requires a negative margin.

## 5. Core Components

### Buttons

- Primary: solid forest green with white text and an optional matching outline icon. Use only one primary button within a given action area.
- Secondary: transparent or paper-white background, thin border, and forest-green text. Match the adjacent primary button's width and corner radius.
- Tertiary: text or icon actions such as “全部记录 ›”. Keep them visually quieter than section headings while preserving adequate touch targets.
- Provide disabled, pressed, and loading states. Preserve button dimensions while loading and prevent duplicate activation.
- Use the danger color and explicit wording for destructive actions, rather than borrowing score colors.

### Lists and Cards

- Present historical games as a continuous list with thin dividers. Do not wrap each row in a large card with its own shadow.
- Place the name and one metadata line on the left, with the score on the right. Maintain consistent baselines, right edges, and vertical rhythm.
- Avoid repeating labels such as “我的分值” when the context already explains the value.
- Use cards for current-game summaries, distinct groups, or independent states. Do not turn every text block into a card.

### Icons and Illustrations

- Use one outline icon family, generally 20–24px with a visual stroke width of approximately 1.5–2px.
- Do not mix heavy filled icons, emoji, thin outline icons, and skeuomorphic icons.
- Card suits and outlined playing cards may provide small brand accents. They must not compete with headings or primary buttons.
- Integrate illustrations naturally into the background; avoid rectangular image stickers with visible white backgrounds. Do not add decorative illustrations to task-dense pages.

### Bottom Navigation

- Keep three equally spaced destinations: `牌局 / 战绩 / 我的`.
- Use an approximately 56px content area plus the actual device bottom safe-area inset. Do not hardcode a device-specific safe-area value.
- Icons are approximately 22px, with approximately 4px between icon and label. Match the visual weight of all three icons.
- Use forest-green icons and text for the selected item, optionally with a short underline below the label. Use the secondary text color for inactive items.
- Coordinate the background with the page and use only a subtle top divider. Avoid large selected pills, heavy shadows, and floating docks.
- Prefer native WeChat navigation capabilities. Custom navigation must correctly handle the capsule area, safe areas, and back navigation. Status bars and close symbols in generated images are illustrative; actual platform components take precedence.

## 6. Homepage Application

When there is no current game, arrange content from top to bottom:

1. Native navigation and a compact brand mark.
2. Headline “来一局，记下好时光” and supporting text “和朋友一起，轻松记分。” Allow natural wrapping on narrow screens.
3. A low-emphasis status, “尚无进行中的牌局”, without a dedicated large card.
4. A solid “扫码加入牌局” button and an outlined “创建牌局” button.
5. The “最近牌局” section and “全部记录” action. Use a continuous history list emphasizing names and scores.
6. Fixed bottom navigation.

When a current game exists, make the scoreboard and “记一笔” the main content. Reuse the same background, typography, dividers, and button system. There is no need to retain the welcome copy or illustration from the no-current-game state.

## 7. Application to Other Pages

| Page                 | Guidance                                                                                                                                                                                     |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Current game         | Emphasize scores and “记一笔”. Keep management actions secondary and recent scoring entries compact.                                                                                         |
| Record a score       | Keep participant selection, input, preview, and confirmation on one page. Use consistent selected states and control radii. Keep results and submission reachable when the keyboard appears. |
| Performance          | Prioritize numbers and trends. Group statistics through layout, reducing separate small cards. Use consistent semantic colors and subtle gridlines in charts.                                |
| History details      | Reuse the history list's hierarchy for names, metadata, and scores. Present long transaction histories in a clear sequence.                                                                  |
| Settlement           | Make final scores the visual focus. One restrained commemorative symbol is acceptable. Use the standard primary-button style for sharing.                                                    |
| Profile              | Use standard grouped lists with a clear hierarchy for identity and settings. Avoid an oversized profile hero.                                                                                |
| Empty / error states | Use a concise explanation and an action when needed. Explain how to recover from errors; avoid large decorative areas.                                                                       |

## 8. AI Agent Implementation Rules and Acceptance Checks

1. Read this document, the relevant PRD, and the existing pages before modifying UI. Reuse existing tokens and components first.
2. Centralize these values in theme variables or shared styles. Avoid duplicating magic values across pages.
3. Use the high-fidelity mockup to understand the overall direction. This document and real-device behavior govern dimensions, platform behavior, and accessibility. Do not reproduce generated text errors, textures, or inaccurate system icons.
4. Visual refactoring must not change scoring semantics, permissions, feature accessibility, or data content without explicit authorization.
5. Check long nicknames, long game names, large scores, zero scores, empty lists, loading states, errors, keyboard expansion, and bottom safe areas.
6. Review screenshots at target sizes: consistent alignment, unclipped text, clear hierarchy, unobscured content beneath tabs, a reachable final list item, and sufficient touch targets.
7. Compose existing patterns before introducing new ones. If a new visual pattern is necessary, document it here before extending it to other pages to prevent visual drift.

### Patterns to Avoid

- Nested large rounded cards, separate shadows on every row, and excessive pale-green blocks.
- Bold type or brand green applied to all headings, buttons, labels, and body text.
- Multiple illustration styles, icon stroke weights, or font systems on one page.
- Oversized welcome areas, status placeholder cards, and meaningless whitespace that crowd out core actions.
- Copying generated-image physical pixels, hardcoding screen heights, or recreating system navigation chrome.
