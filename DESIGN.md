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

## Typography
- **Display/Hero:** Instrument Serif - gives the preview and docs a sharper, more memorable title treatment without drifting into generic SaaS styling.
- **Body:** Instrument Sans - readable, neutral, and close to the utilitarian tone.
- **UI/Labels:** Instrument Sans - keeps labels and help text calm and legible.
- **Data/Tables:** IBM Plex Mono - tabular feel for hashes, refs, and commit metadata.
- **Code:** JetBrains Mono - for any code snippets or command examples in docs.
- **Loading:** Google Fonts or Bunny Fonts CDN links in preview assets; terminal UI should keep using the user's terminal font.
- **Scale:** 12 / 14 / 16 / 20 / 24 / 32 / 40 px, with the smaller steps reserved for metadata and the larger steps for page headers.

## Color
- **Approach:** Balanced restrained palette.
- **Primary:** #E6A23C - stash accent, inspection affordance, and any action that points back to a paused workflow.
- **Secondary:** #36C2B4 - graph and navigation accent, used for structure rather than urgency.
- **Neutrals:** #0D1117, #111927, #16202B, #253041, #4B5A6B, #93A4B4, #E6EDF3.
- **Semantic:** success #7EE081, warning #E6A23C, error #F46D6D, info #7CB8FF.
- **Dark mode:** Default to dark graphite surfaces with a slight warmth in accent colors. Reduce saturation in large surfaces, keep accent colors vivid only at the point of action.

## Spacing
- **Base unit:** 8px.
- **Density:** Compact, but not cramped.
- **Scale:** 2xs(2) xs(4) sm(8) md(16) lg(24) xl(32) 2xl(48) 3xl(64)

## Layout
- **Approach:** Hybrid.
- **Grid:** 3:7 top split for global/context, then a 72:28 graph/rail split in the main body.
- **Max content width:** Terminal-driven, with the main body centered inside the available viewport.
- **Border radius:** Small hierarchy, e.g. sm:4px, md:8px, lg:12px, full:9999px for chips only.

## Motion
- **Approach:** Minimal-functional.
- **Easing:** enter(ease-out) exit(ease-in) move(ease-in-out)
- **Duration:** micro(50-100ms) short(150-250ms) medium(250-400ms) long(400-700ms)

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-07-06 | Initial design system created | Based on stash-graph inspection goals, Git TUI research, and the current shell layout. |
