from PIL import Image, ImageDraw, ImageFont
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

FONT_CANDIDATES = [
    "C:/Windows/Fonts/arialbd.ttf",
    "/System/Library/Fonts/Supplemental/Arial Bold.ttf",
    "/Library/Fonts/Arial Bold.ttf",
    "/System/Library/Fonts/Helvetica.ttc",
    "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
]


def font(size):
    for path in FONT_CANDIDATES:
        if os.path.exists(path):
            return ImageFont.truetype(path, size)
    return ImageFont.load_default()


def make(size=256):
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    r = int(size * 0.22)
    d.rounded_rectangle((0, 0, size - 1, size - 1), radius=r, fill=(11, 18, 32, 255), outline=(34, 211, 238, 255), width=max(2, size // 40))
    f = font(int(size * 0.46))
    text = 'DL'
    box = d.textbbox((0, 0), text, font=f)
    tw, th = box[2] - box[0], box[3] - box[1]
    d.text(((size - tw) / 2 - box[0], (size - th) / 2 - box[1] + size * 0.03), text, font=f, fill=(248, 250, 252, 255))
    dot = max(5, size // 10)
    dx = size - dot - int(size * 0.10)
    dy = int(size * 0.10)
    d.ellipse((dx, dy, dx + dot, dy + dot), fill=(34, 197, 94, 255))
    return img


sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
base = make(256)
base.save(os.path.join(ROOT, 'desktop', 'assets', 'app.ico'), sizes=[(s, s) for s in sizes])
base.save(os.path.join(ROOT, 'desktop', 'assets', 'app.png'))

build_dir = os.path.join(ROOT, 'desktop', 'build')
os.makedirs(build_dir, exist_ok=True)
make(1024).save(os.path.join(build_dir, 'appicon.png'))
print('ok')
