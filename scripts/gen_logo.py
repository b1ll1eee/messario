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

# Concept "Dialogue + M": two speech bubbles in conversation (the back one
# translucent), the front one carrying the M of Messario as a cut-out.
BACK = "M12 13h26a6 6 0 0 1 6 6v11a6 6 0 0 1-6 6H24l-7 6v-6h-5a6 6 0 0 1-6-6V19a6 6 0 0 1 6-6z"
FRONT = "M26 28h26a6 6 0 0 1 6 6v11a6 6 0 0 1-6 6h-5v6l-7-6H26a6 6 0 0 1-6-6V34a6 6 0 0 1 6-6z"
LETTER_M = "M33 44V34l6 6 6-6v10"
BACK_OPACITY = "0.55"


def bubbles(fg, gap):
    """The two bubbles plus the M. `gap` is the colour of the gap between
    bubbles and of the letter (the tile colour, or white when there is no tile)."""
    return (f'<path d="{BACK}" fill="{fg}" opacity="{BACK_OPACITY}"/>'
            f'<path d="{FRONT}" fill="{fg}" stroke="{gap}" stroke-width="3" stroke-linejoin="round"/>'
            f'<path d="{LETTER_M}" fill="none" stroke="{gap}" stroke-width="3.6" stroke-linecap="round" stroke-linejoin="round"/>')


def mark(fg, bg, size=64):
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="0 0 64 64" role="img" aria-label="Messario">'
            f'<rect width="64" height="64" rx="18" fill="{bg}"/>' + bubbles(fg, bg) + "</svg>")


def mark_mono(color, size=64):
    """Single-colour mark with no tile, for print and coloured backgrounds."""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="0 0 64 64" role="img" aria-label="Messario">'
            + bubbles(color, WHITE) + "</svg>")


def wordmark(fg, bg, width=360, height=96):
    """Mark + 'Messario' for README/site headers. Text uses the UI font stack."""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}" role="img" aria-label="Messario">'
            f'<g transform="translate(8 16)"><rect width="64" height="64" rx="18" fill="{bg}"/>' + bubbles(WHITE, bg) + '</g>'
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
