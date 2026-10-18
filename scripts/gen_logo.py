#!/usr/bin/env python3
"""Generate the Messario brand assets (SVG) from one source of truth.

    python3 scripts/gen_logo.py

Writes brand/*.svg, site/favicon.svg and web/public/favicon.svg. PNG
derivatives (apple-touch-icon, OG image) are rendered from these by
scripts/render_logo.mjs. Re-running produces identical files.
"""
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TEAL = "#0F766E"
TEAL_DARK = "#2FB5A5"
INK = "#16181D"
WHITE = "#FFFFFF"

# A speech bubble with an M cut out of it: concept C from the logo sheet.
BUBBLE = "M14 14h36a6 6 0 0 1 6 6v22a6 6 0 0 1-6 6H26l-8 8v-8h-4a6 6 0 0 1-6-6V20a6 6 0 0 1 6-6z"
LETTER_M = "M21 39V23l11 11 11-11v16"


def mark(fg, bg, tile=True, size=64):
    parts = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="0 0 64 64" role="img" aria-label="Messario">']
    if tile:
        parts.append(f'<rect width="64" height="64" rx="18" fill="{bg}"/>')
    parts.append(f'<path d="{BUBBLE}" fill="{fg}"/>')
    # The letter is drawn in the tile colour so it reads as a cut-out.
    parts.append(f'<path d="{LETTER_M}" fill="none" stroke="{bg if tile else WHITE}" stroke-width="5.5" stroke-linecap="round" stroke-linejoin="round"/>')
    parts.append("</svg>")
    return "".join(parts)


def mark_mono(color, size=64):
    """Single-colour mark with no tile; the M is a true hole (even-odd)."""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="0 0 64 64" role="img" aria-label="Messario">'
            f'<path d="{BUBBLE}" fill="{color}"/>'
            f'<path d="{LETTER_M}" fill="none" stroke="#FFFFFF" stroke-width="5.5" stroke-linecap="round" stroke-linejoin="round"/>'
            "</svg>")


def wordmark(fg, bg, width=360, height=96):
    """Mark + 'Messario' for README/site headers. Text uses the UI font stack."""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}" role="img" aria-label="Messario">'
            f'<g transform="translate(8 16) scale(1)">'
            f'<rect width="64" height="64" rx="18" fill="{bg}"/>'
            f'<path d="{BUBBLE}" fill="{WHITE if bg != "none" else fg}"/>'
            f'<path d="{LETTER_M}" fill="none" stroke="{bg if bg != "none" else WHITE}" stroke-width="5.5" stroke-linecap="round" stroke-linejoin="round"/>'
            f'</g>'
            f'<text x="90" y="64" font-family="Instrument Sans, IBM Plex Sans Thai, system-ui, -apple-system, sans-serif" font-size="44" font-weight="700" letter-spacing="-1" fill="{fg}">Messario</text>'
            "</svg>")


FILES = {
    "brand/messario-mark.svg": mark(WHITE, TEAL),
    "brand/messario-mark-dark.svg": mark(INK, TEAL_DARK),
    "brand/messario-mark-mono.svg": mark_mono(TEAL),
    "brand/messario-wordmark.svg": wordmark(INK, TEAL),
    "brand/messario-wordmark-dark.svg": wordmark("#E9EBEF", TEAL_DARK),
    "site/favicon.svg": mark(WHITE, TEAL),
    "web/public/favicon.svg": mark(WHITE, TEAL),
}

for rel, content in FILES.items():
    path = os.path.join(ROOT, rel)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content + "\n")
    print("wrote", rel)

if "--check" in sys.argv:
    import xml.dom.minidom as m
    for rel in FILES:
        m.parse(os.path.join(ROOT, rel))
    print("all SVGs parse as XML")
