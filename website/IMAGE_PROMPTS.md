# Glovebox Website Image Generation Prompts

These prompts are formatted for the nanobanana Gemini CLI extension. Copy and paste each command block directly into your terminal.

**Color Reference:**
- Teal (primary): #1d7a74
- Warm gray: #6b6561
- Near-black: #1c1c1c
- Cream background: #faf8f5

---

## Hero Illustration

**Output:** `hero-glovebox.png` (will generate 4 variations)

```
/generate "hero-glovebox.png 1950s NASA/Bell Labs technical manual illustration of a scientific glovebox laboratory containment chamber. Cross-section cutaway view showing the interior of a sealed rectangular chamber. Two circular glove ports on the front with thick rubber gloves extending inward. Stylized human hands reaching through the gloves, manipulating abstract geometric shapes inside representing code or data. Include technical annotation labels: the word SAMPLE with a thin leader line pointing to the contents inside, and the word OPERATOR with a leader line pointing to the hands. Add subtle dimension markers and hairline construction lines for engineering authenticity. Clean precise linework, flat design, no gradients, no 3D rendering, no shading. Primary color teal #1d7a74, secondary lines warm gray #6b6561, background solid cream #faf8f5. Serious professional technical drafter aesthetic, not cartoon or atomic-age kitsch." --styles="vintage,minimalist" --count=4 --preview
```

---

## Feature Icons

### Composable Mods Icon

**Output:** `icon-composable-mods.png`

```
/icon "icon-composable-mods.png 1950s electronics schematic style icon showing interlocking modular blocks. Three to four rectangular components of different sizes connected together with visible connector points or notches at their interfaces. Suggests interchangeable composable parts that can be rearranged. Simple geometric shapes with small circles at connection points. Include one or two dashed lines suggesting additional connection options. Clean precise linework in teal #1d7a74 only. Flat design, no gradients, no 3D effects, no shading. Technical engineering diagram aesthetic like vintage Bell Labs schematics." --sizes="64" --style="minimal" --background="transparent" --corners="sharp" --count=4 --preview
```

---

### Layered Images Icon

**Output:** `icon-layered-images.png`

```
/icon "icon-layered-images.png 1950s geological cross-section diagram style icon showing stacked horizontal layers. Three to four layers viewed from the side, bottom layer noticeably thicker representing the base (about 40% of total height), upper layers progressively thinner. Clean vertical cut edges revealing internal layer structure. Each layer clearly separated with visible boundary lines. Slightly staggered edges to distinguish layers. Small arrow annotation on right side pointing to layers. Clean precise linework in teal #1d7a74 only. Flat design, no gradients, no 3D effects, no shading. Technical architectural or geological stratigraphy aesthetic." --sizes="64" --style="minimal" --background="transparent" --corners="sharp" --count=4 --preview
```

---

### Persistent Containers Icon

**Output:** `icon-persistent-containers.png`

```
/icon "icon-persistent-containers.png 1950s industrial equipment schematic style icon showing a durable container with hinged lid. Side cutaway view of rectangular container, lid shown slightly open at 15-20 degrees with visible hinge. Container walls drawn with double lines showing thickness and durability. Inside shows simple stable organized contents (small circles or rectangles). Small curved arrow near lid suggesting open/close motion. Laboratory specimen container or equipment case aesthetic. Clean precise linework in teal #1d7a74 only. Flat design, no gradients, no 3D effects, no shading." --sizes="64,128,256" --style="minimal" --background="transparent" --corners="sharp" --preview
```

---

### Commit Workflow Icon

**Output:** `icon-commit-workflow.png`

```
/icon "icon-commit-workflow.png 1950s industrial process flow diagram style icon showing circular workflow cycle. Circular arrow design with continuous cycle path, two to three nodes or stations along the path represented by small geometric shapes (squares or circles). Bold directional arrows showing clockwise flow. One node includes small checkmark symbol suggesting verification or commit step. Technical precision with clean corners and proper arrowhead proportions. Clean precise linework in teal #1d7a74 only. Flat design, no gradients, no 3D effects, no shading. Factory workflow schematic aesthetic." --sizes="64" --style="minimal" --background="transparent" --corners="sharp" --preview
```

---

## Documentation Page Header

**Output:** `docs-header.png`

```
/generate "docs-header.png Wide horizontal banner 1200x200 pixels, 1950s technical manual aesthetic. Subtle engineering graph paper grid background in light warm gray #6b6561. Left section shows open three-ring binder or technical manual at slight angle with visible binding rings and pages with faint text lines. Center section has small floating technical elements: simplified flowchart with connected boxes, small circuit schematic fragment, simple cross-section diagram - scattered but balanced. Right section shows numbered section markers 01 02 03 in monospace font arranged vertically with small horizontal lines extending from each. Primary elements in teal #1d7a74, background cream #faf8f5. Clean linework, flat design, no gradients, light airy density. Bell Labs technical report title page aesthetic." --styles="vintage,minimalist" --count=4 --preview
```

---

## Footer Decoration

**Output:** `footer-glovebox.png`

```
/icon "footer-glovebox.png Extremely simplified logo-like pictogram of a glovebox, 1950s engineering aesthetic. Simple rounded rectangle representing front face of glovebox chamber. Inside the rectangle two circles positioned side by side representing glove ports. From each circular port a minimal curved shape suggesting gloves extending inward. Highly simplified almost iconic design using only essential lines. No text, no labels, no annotations. Must remain clear and recognizable when scaled to 100px wide. Clean linework in teal #1d7a74 only. Flat design, no gradients, no shading." --sizes="256" --style="minimal" --background="transparent" --corners="rounded" --preview
```

---

## Favicon

**Output:** `favicon.png` (will generate multiple sizes)

```
/icon "favicon.png Extremely minimal glovebox icon that must be recognizable at 16 pixels. Simple rounded rectangle as outer frame representing the chamber. Inside two filled circles side by side representing glove ports, clearly visible and separated even at tiny sizes. Use filled solid shapes not stroked lines for visibility at small sizes. Teal #1d7a74 shapes on cream #faf8f5 or white background. Bold clear shapes that do not blur or merge at 16x16. Maximum simplicity, no fine details, no thin lines." --sizes="16,32,180" --type="favicon" --style="minimal" --background="white" --corners="rounded" --preview
```

---

## Open Graph / Social Preview Image

**Output:** `og-image.png`

```
/generate "og-image.png Social media preview image 1200x630 pixels, 1950s technical manual aesthetic. Left side (45% width): simplified technical illustration of glovebox in cross-section view with rectangular sealed chamber, two circular glove ports with gloves extending inward, abstract geometric shapes inside representing contents, minimal technical annotation lines, subtle engineering grid in background. Right side (55% width): large title text Glovebox in geometric sans-serif font like Jost or Futura in near-black #1c1c1c, below it tagline Sandboxed development environments in warm gray #6b6561. Background solid cream #faf8f5 with optional very subtle grid texture. Clean visual separation between illustration and text. Generous 60px padding from all edges. Professional, readable at thumbnail size. Primary accent teal #1d7a74." --styles="vintage,minimalist" --count=4 --preview
```

---

## 404 Page Illustration

**Output:** `404-illustration.png`

```
/generate "404-illustration.png 1950s technical manual illustration of empty glovebox for 404 error page, 400x300 pixels. Frontal view of sealed glovebox chamber with two circular glove ports visible on front face. The rubber gloves hanging LIMP and EMPTY from the ports, drooping downward in relaxed deflated posture suggesting no operator and nothing happening. Chamber interior visibly EMPTY with no contents. In center of empty chamber a simple question mark in monospace font style. Optional small technical label reading NO SAMPLE DETECTED in tiny text. One or two hairline construction lines for authenticity. Primary lines teal #1d7a74, secondary lines warm gray #6b6561, background cream #faf8f5. Clean linework, flat design, no gradients, no shading. Dry technical humor aesthetic, not cartoonish - matter-of-fact empty workspace feeling." --styles="vintage,minimalist" --count=4 --preview
```

---

## Production Notes

- All images output as PNG format
- Each command generates 4 variations (`--count=4`) for selection
- Images auto-open with `--preview` flag
- After selecting best variation, optimize PNGs with ImageOptim or TinyPNG
- For icons needing transparency, may need post-processing to remove background
