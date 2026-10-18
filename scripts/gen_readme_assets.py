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
    """A conversation that plays itself in the app's bubble style; loops."""
    W, H = 1200, 440
    cycle = 11.0
    grad_id = "own-grad"
    rows = [
        dict(kind="other", ini="ณศ", bg="#fce7f3", fg="#9d174d", name="ณัฐพล ศรีสุข", time="09:31",
             body="เมื่อคืน merge PR ระบบ invoice เข้า main แล้วครับ บ่ายนี้จะ deploy production", w=640),
        dict(kind="own", time="09:34", body="ไม่มีค่ะ ของฝั่งหน้าบ้านเรียบร้อยหมดแล้ว", w=380),
        dict(kind="bot", name="Deploy Bot", time="13:02", body="production v1.4.2 สำเร็จ · 1m 48s · a81f2c9 invoice: fix VAT rounding", w=600),
        dict(kind="other", ini="ณศ", bg="#fce7f3", fg="#9d174d", name="ณัฐพล ศรีสุข", time="13:05",
             body="ขึ้นแล้วครับ ฝากทีมขายช่วยเช็กหน้า invoice ด้วย", w=470),
        dict(kind="own", time="13:07", body="เช็กแล้ว ผ่านค่ะ ✓", w=200),
    ]
    p = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" role="img" '
         f'aria-label="An animated Messario conversation: two teammates and a deploy bot in a group chat.">',
         f'<defs><linearGradient id="{grad_id}" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#2fb5a5"/><stop offset="1" stop-color="#1d8f84"/></linearGradient></defs>',
         f'<rect x="0.5" y="0.5" width="{W-1}" height="{H-1}" rx="22" fill="{t["card"]}" stroke="{t["line"]}"/>',
         # header: stacked avatars + title
         f'<circle cx="46" cy="36" r="17" fill="#dbeafe"/>' + text(t, 46, 41, "ณ", 12, "#1e3a8a", "middle", 800),
         f'<circle cx="68" cy="36" r="17" fill="#fce7f3" stroke="{t["card"]}" stroke-width="2.5"/>' + text(t, 68, 41, "ว", 12, "#9d174d", "middle", 800),
         f'<circle cx="90" cy="36" r="17" fill="{t["soft"]}" stroke="{t["card"]}" stroke-width="2.5"/>' + text(t, 90, 41, "+10", 10, "text", "middle", 800),
         text(t, 120, 32, "ทีมพัฒนา", 16, "text", weight=800),
         f'<circle cx="124" cy="46" r="3.5" fill="#34d399"/>' + text(t, 132, 50, "ออนไลน์ 7 · 12 คน · 2 บอท", 12, "muted", weight=600),
         f'<line x1="1" y1="68" x2="{W-1}" y2="68" stroke="{t["line"]}"/>']
    y = 96
    last_own_y = 0
    for i, r in enumerate(rows):
        t0 = 0.6 + i * 1.6
        kt = f'0;{t0/cycle:.3f};{(t0+0.4)/cycle:.3f};0.93;1'
        g = [f'<g opacity="0"><animate attributeName="opacity" values="0;0;1;1;0" keyTimes="{kt}" dur="{cycle}s" repeatCount="indefinite"/>',
             f'<animateTransform attributeName="transform" type="translate" values="0 8;0 8;0 0;0 0;0 0" keyTimes="{kt}" dur="{cycle}s" repeatCount="indefinite"/>']
        if r["kind"] == "own":
            x = W - 32 - r["w"]
            g.append(text(t, W - 40, y + 4, r["time"], 11.5, "dim", "end", 600))
            w, top, h = r["w"], y + 10, 44
            g.append(f'<path d="M{x+20} {top}H{x+w-20}A20 20 0 0 1 {x+w} {top+20}V{top+h-6}A6 6 0 0 1 {x+w-6} {top+h}H{x+20}A20 20 0 0 1 {x} {top+h-20}V{top+20}A20 20 0 0 1 {x+20} {top}Z" fill="url(#{grad_id})"/>')
            g.append(text(t, x + 18, y + 35, r["body"], 15, "#06201c", weight=600))
            last_own_y = y + 50
            step = 62
        else:
            bot = r["kind"] == "bot"
            # avatar
            if bot:
                g.append(f'<circle cx="49" cy="{y+40}" r="17" fill="{t["tealsoft"]}"/>')
                g.append(f'<g transform="translate(40 {y+31}) scale(0.75)" fill="none" stroke="{t["tealsoftink"]}" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="8" width="16" height="12" rx="4"/><path d="M12 4v4"/><circle cx="9" cy="14" r="1"/><circle cx="15" cy="14" r="1"/></g>')
            else:
                g.append(f'<circle cx="49" cy="{y+40}" r="17" fill="{r["bg"]}"/>' + text(t, 49, y + 45, r["ini"], 12, r["fg"], "middle", 800))
            nx = 78
            g.append(text(t, nx, y + 4, r["name"], 13, "text", weight=700))
            nw = len(r["name"]) * 7.4 + 10
            if bot:
                g.append(f'<rect x="{nx+nw}" y="{y-8}" width="34" height="16" rx="5" fill="{t["tealsoft"]}"/>' + text(t, nx + nw + 17, y + 4, "BOT", 9.5, "tealsoftink", "middle", 800, spacing=0.6))
                nw += 42
            g.append(text(t, nx + nw, y + 4, r["time"], 11.5, "dim", weight=600))
            w, top, h = r["w"], y + 10, 44
            g.append(f'<path d="M{nx+6} {top}H{nx+w-20}A20 20 0 0 1 {nx+w} {top+20}V{top+h-20}A20 20 0 0 1 {nx+w-20} {top+h}H{nx+20}A20 20 0 0 1 {nx} {top+h-20}V{top+6}A6 6 0 0 1 {nx+6} {top}Z" fill="{t["soft"]}"/>')
            g.append(text(t, nx + 16, y + 35, r["body"], 15, "text"))
            step = 70
        g.append("</g>")
        p.extend(g)
        y += step
    # read receipt (avatar stack) under the last own message, late in the cycle
    sx = W - 40
    p.append(f'<g opacity="0"><animate attributeName="opacity" values="0;0;1;1;0" keyTimes="0;0.80;0.84;0.93;1" dur="{cycle}s" repeatCount="indefinite"/>'
             + text(t, sx - 54, last_own_y + 10, "อ่านแล้ว", 11.5, "dim", "end", 600)
             + f'<circle cx="{sx-38}" cy="{last_own_y+6}" r="9" fill="#fce7f3" stroke="{t["card"]}" stroke-width="2"/>' + text(t, sx - 38, last_own_y + 9.5, "ณ", 8, "#9d174d", "middle", 800)
             + f'<circle cx="{sx-24}" cy="{last_own_y+6}" r="9" fill="#dcfce7" stroke="{t["card"]}" stroke-width="2"/>' + text(t, sx - 24, last_own_y + 9.5, "ส", 8, "#14532d", "middle", 800)
             + "</g>")
    # floating composer
    p.append(f'<rect x="24" y="{H-70}" width="{W-48}" height="50" rx="16" fill="{t["bg"]}" stroke="{t["line"]}"/>')
    p.append(text(t, 44, H - 40, "ส่งข้อความถึง ทีมพัฒนา", 14, "dim", weight=500))
    p.append(f'<rect x="{W-24-46}" y="{H-65}" width="40" height="40" rx="13" fill="url(#{grad_id})"/>')
    p.append(f'<path d="M{W-24-46+12} {H-45} h16 m-6 -6 l6 6 -6 6" fill="none" stroke="#06201c" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>')
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
