#!/usr/bin/env python3
"""Generate the README illustrations (SVG, light + dark) from code.

    python3 scripts/gen_readme_assets.py

Writes docs/hero-*.svg (banner), docs/demo.svg (animated conversation) and
docs/flow-*.svg (how a message travels). Re-running produces identical output.
"""
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "docs")
os.makedirs(OUT, exist_ok=True)

SANS = "'Instrument Sans', 'IBM Plex Sans Thai', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif"
MONO = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
THEMES = {
    "light": dict(bg="#F7F8FA", card="#FFFFFF", line="#E3E6EB", soft="#EEF0F4", text="#16181D", muted="#5F6675",
                  dim="#8A919E", teal="#0F766E", tealink="#FFFFFF", tealsoft="#DDF3EC", tealsoftink="#0B5C4D"),
    "dark": dict(bg="#15171C", card="#1C1F26", line="#2B3039", soft="#23272F", text="#E9EBEF", muted="#A3A9B5",
                 dim="#767D8A", teal="#2FB5A5", tealink="#0B1F1C", tealsoft="#163A35", tealsoftink="#9EE3D7"),
}

BACK = "M12 13h26a6 6 0 0 1 6 6v11a6 6 0 0 1-6 6H24l-7 6v-6h-5a6 6 0 0 1-6-6V19a6 6 0 0 1 6-6z"
FRONT = "M26 28h26a6 6 0 0 1 6 6v11a6 6 0 0 1-6 6h-5v6l-7-6H26a6 6 0 0 1-6-6V34a6 6 0 0 1 6-6z"
LETTER_M = "M33 44V34l6 6 6-6v10"


def mark(t, x, y, size):
    s = size / 64
    return (f'<g transform="translate({x} {y}) scale({s})"><rect width="64" height="64" rx="18" fill="{t["teal"]}"/>'
            f'<path d="{BACK}" fill="{t["tealink"]}" opacity="0.55"/>'
            f'<path d="{FRONT}" fill="{t["tealink"]}" stroke="{t["teal"]}" stroke-width="3" stroke-linejoin="round"/>'
            f'<path d="{LETTER_M}" fill="none" stroke="{t["teal"]}" stroke-width="3.6" stroke-linecap="round" stroke-linejoin="round"/></g>')


def text(t, x, y, s, size=14, fill="text", anchor="start", weight=400, font=SANS, spacing=0):
    """`fill` is a theme key, or a literal colour when it starts with '#'."""
    color = fill if fill.startswith("#") else t[fill]
    return (f'<text x="{x}" y="{y}" font-family="{font}" font-size="{size}" font-weight="{weight}" '
            f'fill="{color}" text-anchor="{anchor}" letter-spacing="{spacing}">{s}</text>')


def chip(t, x, y, s):
    w = int(len(s) * 7.2) + 28
    return w, (f'<rect x="{x}" y="{y}" width="{w}" height="28" rx="14" fill="{t["tealsoft"]}"/>'
               + text(t, x + w / 2, y + 18.5, s, 12.5, "tealsoftink", "middle", 600))


# ---------------------------------------------------------------- hero banner
def hero(t):
    W, H = 1200, 300
    p = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" '
         f'aria-label="Messario: self-hosted team chat for organizations.">',
         f'<rect width="{W}" height="{H}" rx="20" fill="{t["bg"]}"/>',
         f'<rect x="0.5" y="0.5" width="{W-1}" height="{H-1}" rx="20" fill="none" stroke="{t["line"]}"/>',
         mark(t, 64, 70, 96),
         text(t, 184, 118, "Messario", 54, "text", weight=700, spacing=-2),
         text(t, 184, 158, "Self-hosted team chat for organizations", 22, "muted", weight=500),
         text(t, 184, 188, "แชทองค์กรที่ติดตั้งในเซิร์ฟเวอร์ของคุณเอง · ข้อมูลไม่ออกจากบริษัท", 17, "dim")]
    x = 184
    for s in ["One Go binary + PostgreSQL", "Telegram-style bot API", "Read receipts · files · channels", "Offline license keys"]:
        w, c = chip(t, x, 214, s)
        p.append(c)
        x += w + 10
    # subtle decorative bubbles on the right
    for i, (bx, by, bs, op) in enumerate([(930, 54, 150, 0.10), (1010, 128, 110, 0.16), (880, 170, 80, 0.08)]):
        s = bs / 64
        d = FRONT if i % 2 else BACK
        p.append(f'<path d="{d}" transform="translate({bx} {by}) scale({s})" fill="{t["teal"]}" opacity="{op}"/>')
    p.append("</svg>\n")
    return "\n".join(p)


# ---------------------------------------------------------------- animated demo
def demo(t):
    """A conversation that plays itself: messages fade in one after another, then loops."""
    W, H = 1200, 420
    rows = [("ณศ", "ณัฐพล ศรีสุข", "09:31", "เมื่อคืน merge PR ระบบ invoice เข้า main แล้วครับ บ่ายนี้จะ deploy production", "#dce6f2", "#26405e", False),
            ("วท", "วริศรา ทองดี", "09:34", "ไม่มีค่ะ ของฝั่งหน้าบ้านเรียบร้อยหมดแล้ว", "#e9e2f5", "#43306b", False),
            ("", "Deploy Bot", "13:02", "production v1.4.2 สำเร็จ · 1 นาที 48 วิ · a81f2c9 invoice: fix VAT rounding", "", "", True),
            ("ณศ", "ณัฐพล ศรีสุข", "13:05", "ขึ้นแล้วครับ ฝากทีมขายช่วยเช็กหน้า invoice ด้วย", "#dce6f2", "#26405e", False),
            ("วท", "วริศรา ทองดี", "13:07", "เช็กแล้ว ผ่านค่ะ ✓", "#e9e2f5", "#43306b", False)]
    cycle = 11.0
    p = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" '
         f'aria-label="An animated Messario conversation: two teammates and a deploy bot in a group chat.">',
         f'<rect x="0.5" y="0.5" width="{W-1}" height="{H-1}" rx="16" fill="{t["card"]}" stroke="{t["line"]}"/>',
         f'<rect x="1" y="1" width="{W-2}" height="56" rx="16" fill="{t["bg"]}"/>',
         f'<rect x="1" y="40" width="{W-2}" height="17" fill="{t["bg"]}"/>',
         f'<line x1="1" y1="57" x2="{W-1}" y2="57" stroke="{t["line"]}"/>',
         f'<rect x="24" y="12" width="34" height="34" rx="10" fill="#dce6f2"/>',
         text(t, 41, 34, "ที", 13, "#26405e", "middle", 600),
         text(t, 70, 29, "ทีมพัฒนา", 15, "text", weight=700),
         text(t, 70, 46, "4 คน · 1 บอท", 12, "dim")]
    y = 92
    for i, (ini, name, time, body, bg, fg, bot) in enumerate(rows):
        t0 = 0.6 + i * 1.6
        g = [f'<g opacity="0">',
             f'<animate attributeName="opacity" values="0;0;1;1;0" keyTimes="0;{t0/cycle:.3f};{(t0+0.4)/cycle:.3f};0.93;1" dur="{cycle}s" repeatCount="indefinite"/>',
             f'<animateTransform attributeName="transform" type="translate" values="0 8;0 8;0 0;0 0;0 0" keyTimes="0;{t0/cycle:.3f};{(t0+0.4)/cycle:.3f};0.93;1" dur="{cycle}s" repeatCount="indefinite"/>']
        if bot:
            g.append(f'<rect x="24" y="{y-20}" width="30" height="30" rx="9" fill="{t["tealsoft"]}"/>')
            g.append(f'<g transform="translate(31 {y-13}) scale(0.25)" fill="none" stroke="{t["tealsoftink"]}" stroke-width="7" stroke-linecap="round" stroke-linejoin="round">'
                     f'<rect x="4" y="8" width="56" height="44" rx="12"/><path d="M32 -4v12"/><circle cx="22" cy="30" r="3"/><circle cx="42" cy="30" r="3"/></g>')
            g.append(text(t, 68, y - 2, name, 14, "text", weight=600))
            g.append(f'<rect x="{68 + 7.6*len(name) + 10}" y="{y-14}" width="32" height="16" rx="4" fill="{t["tealsoft"]}"/>')
            g.append(text(t, 68 + 7.6*len(name) + 26, y - 2, "BOT", 10, "tealsoftink", "middle", 700, spacing=0.8))
            g.append(text(t, 68 + 7.6*len(name) + 52, y - 2, time, 12, "dim"))
            g.append(f'<rect x="68" y="{y+8}" width="700" height="40" rx="10" fill="{t["bg"]}" stroke="{t["line"]}"/>')
            g.append(f'<rect x="68" y="{y+8}" width="3" height="40" rx="1.5" fill="{t["teal"]}"/>')
            g.append(f'<circle cx="88" cy="{y+28}" r="4" fill="#2E7D4F"/>')
            g.append(text(t, 100, y + 33, body, 14, "text", weight=500))
            step = 70
        else:
            g.append(f'<rect x="24" y="{y-20}" width="30" height="30" rx="9" fill="{bg}"/>')
            g.append(text(t, 39, y, ini, 12, fg, "middle", 600))
            g.append(text(t, 68, y - 2, name, 14, "text", weight=600))
            g.append(text(t, 68 + 7.6*len(name) + 10, y - 2, time, 12, "dim"))
            g.append(text(t, 68, y + 22, body, 15, "text"))
            step = 56
        g.append("</g>")
        p.extend(g)
        y += step
    # read receipt under the last message, late in the cycle
    p.append(f'<g opacity="0"><animate attributeName="opacity" values="0;0;1;1;0" keyTimes="0;0.80;0.84;0.93;1" dur="{cycle}s" repeatCount="indefinite"/>'
             f'<path d="M70 {y-2} l4 4 8-8" fill="none" stroke="{t["dim"]}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>'
             + text(t, 88, y + 2, "อ่านแล้ว 3 จาก 3", 12, "dim") + "</g>")
    # composer
    p.append(f'<rect x="24" y="{H-64}" width="{W-48}" height="44" rx="12" fill="{t["card"]}" stroke="{t["line"]}"/>')
    p.append(text(t, 42, H - 37, "ส่งข้อความถึง ทีมพัฒนา", 14, "dim"))
    p.append(f'<rect x="{W-24-40}" y="{H-60}" width="36" height="36" rx="9" fill="{t["teal"]}"/>')
    p.append(f'<path d="M{W-24-40+11} {H-42} h14 m-5 -5 l5 5 -5 5" fill="none" stroke="{t["tealink"]}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>')
    p.append("</svg>\n")
    return "\n".join(p)


# ---------------------------------------------------------------- how a message travels
def flow(t):
    W, H = 1200, 230
    p = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" '
         f'aria-label="How a message travels: client sends over WebSocket, the core assigns the next sequence number in PostgreSQL, then fans out to members, bot webhooks and connected devices.">',
         f'<rect x="0.5" y="0.5" width="{W-1}" height="{H-1}" rx="16" fill="{t["card"]}" stroke="{t["line"]}"/>']
    nodes = [(120, "Client", "Vue · outbox", "client_msg_id"), (360, "WebSocket / REST", "auth · origin check", "send"),
             (600, "Core", "use cases · ports", "seq = last_seq + 1"), (840, "PostgreSQL", "messages · read_seq", "gap-free"),
             (1080, "Fan-out", "devices · bot webhooks", "msg · read")]
    Y = 96
    rail = f"M {nodes[0][0]+70} {Y} L {nodes[-1][0]-70} {Y}"
    p.append(f'<path d="{rail}" stroke="{t["line"]}" stroke-width="1.5"/>')
    for x, n, sub, tag in nodes:
        hi = n == "Core"
        p.append(f'<rect x="{x-70}" y="{Y-30}" width="140" height="60" rx="12" fill="{t["bg"] if not hi else t["tealsoft"]}" stroke="{t["teal"] if hi else t["line"]}" stroke-width="1.5"/>')
        p.append(text(t, x, Y - 4, n, 14, "tealsoftink" if hi else "text", "middle", 700))
        p.append(text(t, x, Y + 16, sub, 11, "tealsoftink" if hi else "dim", "middle", font=MONO))
        p.append(text(t, x, Y + 56, tag, 12, "muted", "middle", 500, font=MONO))
    p.append(f'<circle r="4" fill="{t["teal"]}"><animateMotion dur="3.2s" repeatCount="indefinite" path="{rail}"/>'
             f'<animate attributeName="opacity" values="0;1;1;0" keyTimes="0;0.1;0.9;1" dur="3.2s" repeatCount="indefinite"/></circle>')
    p.append(text(t, 48, H - 36, "Retries are idempotent (same client_msg_id → same message). After a reconnect a client asks for everything after the last seq it holds.", 13, "muted"))
    p.append("</svg>\n")
    return "\n".join(p)


for name, t in THEMES.items():
    for key, fn in [("hero", hero), ("demo", demo), ("flow", flow)]:
        with open(os.path.join(OUT, f"{key}-{name}.svg"), "w", encoding="utf-8") as f:
            f.write(fn(t))
        print("wrote", f"docs/{key}-{name}.svg")
