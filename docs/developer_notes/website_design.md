# Glovebox Website Design Reference

This document captures the design decisions for the Glovebox project website.

## Tech Stack

- **Framework:** Astro with Tailwind CSS
- **Hosting:** GitHub Pages
- **CI/CD:** GitHub Actions
- **Location:** `/website` folder in the main repository

## Design Philosophy

**Modern structure with 1950s scientific accents.**

The aesthetic draws from early NASA, Bell Labs, and Space Race era engineering - serious professionals solving impossible problems. Not atomic-age kitsch, but the precision and confidence of a well-written technical manual.

**Key metaphor:** The website is the clean, well-lit laboratory (warm white, organized, precise). Terminal/code blocks are the glovebox itself (dark, contained, where the real work happens). The dark code examples sit within the light page like a physical glovebox mounted in a sterile lab wall.

**Target audience:** Terminal-native developers who prefer vim/emacs over VS Code, live in tmux/zellij, and want their environment containerized without friction.

---

## Color Palette

| Role            | Color         | Hex       |
|-----------------|---------------|-----------|
| Background      | Warm cream    | `#faf8f5` |
| Body text       | Near-black    | `#1c1c1c` |
| Muted text      | Warm gray     | `#6b6561` |
| Terminal bg     | Deep charcoal | `#0f1115` |
| Terminal text   | Off-white     | `#e6e4e0` |
| Primary accent  | Instrument teal | `#1d7a74` |
| Callout/warning | Muted red     | `#b84c3f` |

---

## Typography

| Role      | Font              | Notes                              |
|-----------|-------------------|------------------------------------|
| Headlines | Jost              | Geometric, Futura lineage, free    |
| Body      | Jost              | Same family for cohesion           |
| Code      | JetBrains Mono    | Crisp, dev-friendly, has character |

---

## Texture & Layout

- **Flat/clean** - no paper grain textures
- **Hairline rules** - `1px` warm gray borders for structure
- **Generous whitespace** - let the content breathe
- **Grid accents** - subtle engineering-paper grid lines in hero background or as section dividers

---

## Component Patterns

### Code Blocks
- Dark terminal style (`#0f1115` background)
- Teal for prompt/commands
- Slight border-radius (`4px`)
- Subtle shadow to lift off the cream background

### Feature Cards
- White background (`#ffffff`) on cream page
- Hairline border
- Small teal accent (top border or icon)

### Buttons
- Primary: Teal background, cream text
- Secondary: Outlined in teal

### Section Headers
- Number in teal (`01.`)
- Title in near-black
- Hairline rule beneath

---

## Glovebox Illustration

**Style:** Technical diagram / cross-section view

- Clean, consistent line weight
- Cross-section of a scientific glovebox
- Two glove ports clearly visible
- Minimal labels (e.g., "SAMPLE" inside, "OPERATOR" outside)
- Could show stylized hands reaching in

Used in: Hero image, possibly footer, favicon derivative

---

## Site Structure

```
/                       Landing page
/docs/                  Documentation hub
/docs/getting-started   Installation & first run
/docs/commands          Command reference
/docs/mods              Available mods
/docs/custom-mods       Creating your own
/docs/architecture      How it works
/docs/workflows         Usage patterns
/docs/configuration     Profiles & env vars
```

---

## Landing Page Content

### Hero

> **AI assistants run code. So does npm install.**
>
> Glovebox gives you a Docker sandbox that actually feels like home. Your shell, your editor, your workflow—running safely inside a container with your project mounted.

```bash
brew install joelhelbling/glovebox/glovebox
```

*[Get Started]*

### Why Glovebox

> AI coding assistants are powerful—but they run code. So do npm packages, pip installs, and that shell script you found on Stack Overflow. Running untrusted code on your development machine is a calculated risk.
>
> You could spin up a VM. You could fight with container configs every time. But that kills your flow.
>
> Glovebox is a sandboxed Docker environment that actually feels like yours. Configure it once with the shell, editor, and tools you want. Run it in any project. Your environment travels with you, safely isolated from your host machine.
>
> Think of it as glamping on Jurassic Island: even in mortal danger, you still get your Nespresso.

### Features (4 cards)

**Composable Mods**
Mix and match shells, editors, languages, and AI tools. Build exactly the environment you want from reusable pieces.

**Layered Images**
Build your base environment once. Extend it per-project with additional tools. No redundant rebuilds.

**Persistent Containers**
Your changes survive between sessions. Install something ad-hoc, it's still there tomorrow.

**Commit Workflow**
Made changes you want to keep? Glovebox detects them and offers to commit them back to the image.

### Quick Start

**One-time setup:**

```bash
glovebox init --base    # Select your OS, shell, editor, tools
glovebox build --base   # Build the base image
```

**Then, in any project:**

```bash
cd ~/projects/my-app
glovebox run
```

You're inside a sandboxed container. Your project is mounted at `/workspace`. Your shell, your editor, your tools—all there. When you exit, your container persists. When you return, it's waiting.

**Clean up when needed:**

```bash
glovebox clean --all
```

### Is This For Me?

**Glovebox is for you if:**

- You run AI coding assistants and want to limit the blast radius
- You connect MCP servers or other tools that reach into your filesystem
- You evaluate npm packages, pip installs, or random scripts before trusting them
- You prefer vim, emacs, or neovim over VS Code
- You think in tmux panes or zellij tabs
- You want consistent environments across projects without VM overhead
- You're a hacker (in the good, MIT sense) who experiments with hazardous things

**Glovebox is not:**

- Infrastructure for production environments
- A security solution for deployed code
- A replacement for proper sandboxing in CI/CD
- A GUI-first experience

Glovebox is a personal workbench tool. It doesn't go in your code and doesn't run on your server. It's the sealed chamber on your workbench where you safely handle the unknown.

### Footer

```
GitHub · Docs · MIT License

From the workbench of Joel Helbling

[glovebox emoji] Now get in there and do some science!
```

---

## Documentation

### Structure

Numbered sections (technical manual aesthetic):

```
01. Getting Started     Installation and first run
02. Commands            The complete command reference
03. Mods                Available mods and how they compose
04. Custom Mods         Creating your own mods
05. Architecture        How layered images and persistence work
06. Workflows           Common usage patterns
07. Configuration       Profiles and environment variables
```

### Docs Hub (`/docs`)

> **Operating Manual**
>
> Everything you need to run the equipment.

Card grid linking to each numbered section.

### Page Template

- Section number + title in header ("03. Mods")
- Prev/Next navigation at bottom
- Sidebar showing all sections with current highlighted
- Clean typography, dark code blocks

---

## Repository Structure

```
/glovebox
  ├── cmd/                    # CLI commands
  ├── internal/               # Go packages
  ├── docs/                   # Developer notes (not user docs)
  ├── website/                # Astro project
  │   ├── src/
  │   │   ├── content/
  │   │   │   └── docs/       # User documentation (markdown)
  │   │   ├── pages/
  │   │   └── components/
  │   └── ...
  ├── README.md               # Links to website for docs
  └── ...
```

User-facing documentation lives in `/website/src/content/docs/`. The existing `/docs` folder remains for developer/contributor notes. README links to the website for user documentation.

---

## Implementation Notes

### GitHub Actions

Build and deploy to GitHub Pages on push to main.

### Astro Configuration

- Content collections for docs
- Tailwind for styling
- Static output for GitHub Pages

### Migration

1. Create `/website` Astro project
2. Move user documentation from `/docs/*.md` to `/website/src/content/docs/`
3. Add frontmatter to docs (title, order, description)
4. Update README to link to website for documentation
5. Keep `/docs/developer_notes/` for contributor documentation
