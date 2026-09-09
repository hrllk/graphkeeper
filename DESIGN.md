# Design System - Graphkeeper

## Product Context
- **What this is:** A graph-first Git TUI for reading repository history, branch topology, and stash state quickly.
- **Who it's for:** Developers who need to understand where work paused, where it came from, and how to continue it safely.
- **Space/industry:** Developer tools, terminal apps, Git navigation.
- **Project type:** Terminal app with supporting preview/docs surfaces.

## Aesthetic Direction
- **Direction:** Industrial/Utilitarian with an editorial edge.
- **Decoration level:** Minimal.
- **Mood:** Precise, quiet, and information-dense. The interface should feel like a map for repository history, not a general-purpose Git cockpit.
- **Reference sites:** https://github.com/jesseduffield/lazygit, https://github.com/gitui-org/gitui, https://jonas.github.io/tig/, https://github.com/git-up/GitUp

---

# Terminal UI

**This section governs the app.** Everything below the "Preview and docs surfaces"
heading governs the README, preview images and any web surface, and does not
apply to the terminal.

The split matters because `CLAUDE.md` tells the agent to read this file before
any visual decision and to flag code that does not match it. When this file
described a hex palette the terminal cannot emit, following that instruction
made every colour in the app a false positive.

## Color

**Contract: ANSI 0-15 and terminal attributes only.** No RGB, no ANSI-256.
`docs/highlighting-color-map.md` holds the token table; this is the rule behind
it, and `internal/app/theme.go` is the single place allowed to build a colour.

ANSI colour numbers are palette slots, not fixed RGB values. The user's terminal
decides what slot 3 looks like, which is the point: the app inherits the theme
the user already chose rather than fighting it.

| Role | Expression |
|---|---|
| section/navigation heading | ANSI blue + bold |
| keyboard shortcut | ANSI magenta + bold |
| loading, warning, stash | ANSI yellow + bold |
| success, current, HEAD | ANSI green + bold |
| dirty, error, conflict | ANSI red + bold |
| remote/target context | ANSI blue |
| tag, provenance | ANSI magenta |
| secondary/help text | terminal default foreground |
| focus and selection | reverse, or underline, written as attributes |

**Every colour needs a non-colour partner.** A signal carried by colour alone
disappears under `NO_COLOR`, on a monochrome terminal, and for a reader who
cannot separate the two hues the palette happened to pick. Warning and error
both land on their ANSI slot and stay distinguishable through visible copy.

**Attributes are not colour.** no-color.org governs colour, so reverse and
underline may be written directly when `NO_COLOR` is set. This is not a
loophole; it is the only way a cursor stays visible there, because lipgloss
selects the Ascii profile under `NO_COLOR` and drops attributes along with
colour. `cursorSignal` in `internal/app/commit_inspector.go` is the shared
implementation, used by both the graph and the right rail.

## Layout

- **Body:** two columns. Full-height `Graph` on the left, and a right rail
  stacking `Details`, `Local`, `Remote`, `Tags`.
- **Split:** roughly 72:28 graph to rail. Measured at an 80-column terminal:
  graph 43, rail 17.
- **Width:** terminal-driven and centred, and the frame is always inside the
  screen. Verified from 1 column upward, not only at usable widths.
- **Graph row columns:** commit, branches, state, graph, date, title. State,
  graph and date are sized from what the graph actually contains and disappear
  when they would earn nothing; the title takes whatever remains. See
  `docs/decisions.md` 2026-09-10.

## Density and spacing

- Compact, not cramped. A cell earns its columns or it is not drawn.
- Truncation uses a visible marker, and there is exactly one: `…`. A silently
  hard-cut string reads as broken rather than shortened. The single exception is
  a commit hash, which a reader already reads as an abbreviation.
- Popups name their own keys; `esc` closes everything and is not advertised on a
  second row.

## Line vocabulary

Two glyphs, told apart by position:

| Glyph | Position | Meaning |
|---|---|---|
| `•` | starts a line (indentation allowed) | one list item |
| `·` | within a line | joins peers on that line |

And two drawing vocabularies, which never take each other's job:

| Vocabulary | Draws |
|---|---|
| box drawing (`╭─│╰`) | the app's frame -- borders, pane splits, rules |
| ASCII (`* \| / \\`) | repository history, because that is git's own topology notation |

`internal/app/vocabulary_test.go` enforces both by walking the package's string
literals. See `docs/decisions.md` 2026-09-10.

## Motion

Minimal-functional. A terminal has no easing curves, so motion here means
progress indication and nothing else: show that work is in flight, and stop.

---

# Preview and docs surfaces

**None of this applies to the terminal.** It governs the README, preview images,
and any web page for the project. It is kept because the project has those
surfaces, and separated because a px scale and a border radius are not things a
terminal can honour.

## Typography (preview/docs only)
- **Display/Hero:** Instrument Serif.
- **Body / UI / Labels:** Instrument Sans.
- **Data/Tables:** IBM Plex Mono, for hashes, refs and commit metadata.
- **Code:** JetBrains Mono.
- **Loading:** Google Fonts or Bunny Fonts CDN links in preview assets. The
  terminal uses whatever font the user's terminal is set to.
- **Scale:** 12 / 14 / 16 / 20 / 24 / 32 / 40 px.

## Color (preview/docs only)
- **Primary:** #E6A23C - stash accent and inspection affordance.
- **Secondary:** #36C2B4 - graph and navigation accent.
- **Neutrals:** #0D1117, #111927, #16202B, #253041, #4B5A6B, #93A4B4, #E6EDF3.
- **Semantic:** success #7EE081, warning #E6A23C, error #F46D6D, info #7CB8FF.

These hex values are the intent the ANSI slots gesture at. They are not what the
app emits, and code that uses them is a defect.

## Spacing, radius, motion (preview/docs only)
- **Base unit:** 8px. **Scale:** 2xs(2) xs(4) sm(8) md(16) lg(24) xl(32) 2xl(48) 3xl(64).
- **Border radius:** sm 4px, md 8px, lg 12px, full 9999px for chips.
- **Easing:** enter(ease-out) exit(ease-in) move(ease-in-out).
  **Duration:** micro 50-100ms, short 150-250ms, medium 250-400ms, long 400-700ms.

---

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-07-06 | Initial design system created | Based on stash-graph inspection goals, Git TUI research, and the shell layout at the time. |
| 2026-08-01 | alpha.5/alpha.6 shell redesign | The `3:7 top split for global/context` this file used to describe was removed. Graph became the full-height primary surface with a stacked right rail; the `Global` and `Context Actions` panels moved into the `?` overlay. See `docs/decisions.md` 2026-08-01. |
| 2026-09-10 | Terminal and preview sections separated; Color rewritten as the ANSI contract | This file specified a hex palette while the code emits ANSI 0-15 only, and `CLAUDE.md` directs the agent here before every visual decision. Measured: the rendered shell emits zero `38;2` and zero `38;5` sequences. Typography, radius and easing moved under preview because a terminal cannot honour them. Task 6.1. |
| 2026-09-10 | One truncation marker, and one job per line glyph | The app had four truncation conventions and two glyphs doing four separator jobs; three could appear on one screen. Now `…` marks every cut, `•` starts a line, `·` joins within one, box drawing frames and ASCII draws history. Enforced by a test, not a convention. Task 6.6. |
| 2026-09-10 | Graph row columns are measured, not budgeted for the worst case | State, graph and date size from the graph's actual content; the title takes the remainder. See `docs/decisions.md` 2026-09-10. Task 6.4, 11.9. |
