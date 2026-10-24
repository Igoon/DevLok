from PIL import Image, ImageDraw, ImageFont
import math
import os

W, H = 1080, 1920
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT_DIR = os.path.join(ROOT, "docs", "poster")

BOLD_CANDIDATES = [
    "C:/Windows/Fonts/arialbd.ttf",
    "/System/Library/Fonts/Supplemental/Arial Bold.ttf",
    "/Library/Fonts/Arial Bold.ttf",
    "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
]
REG_CANDIDATES = [
    "C:/Windows/Fonts/arial.ttf",
    "/System/Library/Fonts/Supplemental/Arial.ttf",
    "/Library/Fonts/Arial.ttf",
    "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
]

BG_TOP = (11, 18, 32)
BG_BOTTOM = (8, 13, 24)
CYAN = (34, 211, 238)
GREEN = (34, 197, 94)
WHITE = (248, 250, 252)
MUTED = (148, 163, 184)
DIM = (100, 116, 139)
PANEL = (17, 24, 39)
BORDER = (30, 41, 59)


def font(candidates, size):
    for path in candidates:
        if os.path.exists(path):
            return ImageFont.truetype(path, size)
    return ImageFont.load_default()


def rounded(draw, box, radius, fill=None, outline=None, width=1):
    draw.rounded_rectangle(box, radius=radius, fill=fill, outline=outline, width=width)


def center_text(draw, top, text, f, fill):
    box = draw.textbbox((0, 0), text, font=f)
    w = box[2] - box[0]
    x = (W - w) / 2 - box[0]
    draw.text((x, top - box[1]), text, font=f, fill=fill)
    return box[3] - box[1]


def make():
    img = Image.new("RGB", (W, H), BG_TOP)
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / (H - 1)
        r = int(BG_TOP[0] + (BG_BOTTOM[0] - BG_TOP[0]) * t)
        g = int(BG_TOP[1] + (BG_BOTTOM[1] - BG_TOP[1]) * t)
        b = int(BG_TOP[2] + (BG_BOTTOM[2] - BG_TOP[2]) * t)
        d.line([(0, y), (W, y)], fill=(r, g, b))

    glow = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    cx, cy = W // 2, 430
    for i in range(60, 0, -1):
        rad = 150 + i * 7
        gd.ellipse((cx - rad, cy - rad, cx + rad, cy + rad), fill=(34, 211, 238, 3))
    img = Image.alpha_composite(img.convert("RGBA"), glow).convert("RGB")
    d = ImageDraw.Draw(img)

    eyebrow = font(BOLD_CANDIDATES, 26)
    center_text(d, 120, "L O C A L   D E V   E N V I R O N M E N T", eyebrow, (125, 211, 252))

    icon = 300
    ix = (W - icon) // 2
    iy = 210
    rounded(d, (ix, iy, ix + icon, iy + icon), 70, fill=(11, 18, 32), outline=CYAN, width=9)
    dl = font(BOLD_CANDIDATES, 150)
    box = d.textbbox((0, 0), "DL", font=dl)
    tw, th = box[2] - box[0], box[3] - box[1]
    d.text((ix + (icon - tw) / 2 - box[0], iy + (icon - th) / 2 - box[1] + 6), "DL", font=dl, fill=WHITE)
    dot = 56
    dx = ix + icon - dot - 26
    dy = iy + 26
    d.ellipse((dx, dy, dx + dot, dy + dot), fill=GREEN)

    headline = font(BOLD_CANDIDATES, 148)
    center_text(d, 585, "DevLok", headline, WHITE)

    sub = font(REG_CANDIDATES, 50)
    center_text(d, 770, "Local development, simplified", sub, (203, 213, 225))
    plat = font(BOLD_CANDIDATES, 38)
    center_text(d, 842, "Windows  •  macOS (Intel & Apple Silicon)", plat, MUTED)

    term_w, term_h = 780, 96
    tx = (W - term_w) // 2
    ty = 940
    rounded(d, (tx, ty, tx + term_w, ty + term_h), 24, fill=PANEL, outline=BORDER, width=2)
    mono = font(BOLD_CANDIDATES, 40)
    d.text((tx + 34, ty + 26), "$", font=mono, fill=GREEN)
    d.text((tx + 74, ty + 26), "dl up", font=mono, fill=WHITE)
    run = font(REG_CANDIDATES, 30)
    d.text((tx + term_w - 210, ty + 36), "running", font=run, fill=GREEN)

    feats = [
        "One-click Nginx + PHP + MySQL stack",
        "Runtime pinned per project",
        "Automatic local HTTPS (trusted CA)",
        "App & runtime updates in-app / CLI",
    ]
    chip_w, chip_h, gap = 860, 88, 22
    cxp = (W - chip_w) // 2
    cy0 = 1120
    chip_font = font(REG_CANDIDATES, 34)
    for i, text in enumerate(feats):
        y = cy0 + i * (chip_h + gap)
        rounded(d, (cxp, y, cxp + chip_w, y + chip_h), 20, fill=(15, 23, 42), outline=BORDER, width=2)
        d.ellipse((cxp + 34, y + chip_h // 2 - 11, cxp + 56, y + chip_h // 2 + 11), fill=CYAN)
        box = d.textbbox((0, 0), text, font=chip_font)
        d.text((cxp + 86, y + (chip_h - (box[3] - box[1])) / 2 - box[1]), text, font=chip_font, fill=(226, 232, 240))

    d.line([(W // 2 - 60, 1740), (W // 2 + 60, 1740)], fill=CYAN, width=3)
    foot = font(BOLD_CANDIDATES, 34)
    center_text(d, 1782, "AI-powered by DeepSeek-v4-Flash", foot, (203, 213, 225))
    site = font(REG_CANDIDATES, 28)
    center_text(d, 1836, "github.com/Igoon/DevLok", site, DIM)

    os.makedirs(OUT_DIR, exist_ok=True)
    out = os.path.join(OUT_DIR, "devlok-poster-9x16.png")
    img.save(out)
    print("saved", out, img.size)


if __name__ == "__main__":
    make()
