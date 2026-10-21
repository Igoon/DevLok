from PIL import Image, ImageDraw, ImageFont

def make(size=256):
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    r = int(size * 0.22)
    d.rounded_rectangle((0, 0, size - 1, size - 1), radius=r, fill=(11, 18, 32, 255), outline=(34, 211, 238, 255), width=max(2, size // 40))
    # terminal caret
    cx = int(size * 0.20)
    cy = int(size * 0.43)
    w = max(2, size // 28)
    d.line([(cx, cy), (cx + size * 0.13, cy + size * 0.10), (cx, cy + size * 0.20)], fill=(34, 211, 238, 255), width=w, joint='curve')
    # D monogram
    font_path = 'C:/Windows/Fonts/arialbd.ttf'
    font = ImageFont.truetype(font_path, int(size * 0.62))
    text = 'D'
    box = d.textbbox((0, 0), text, font=font)
    tw, th = box[2] - box[0], box[3] - box[1]
    d.text((int(size * 0.53) - tw / 2, int(size * 0.50) - th / 2 - box[1]), text, font=font, fill=(248, 250, 252, 255))
    # status dot
    dot = max(5, size // 9)
    dx = size - dot - int(size * 0.13)
    dy = int(size * 0.13)
    d.ellipse((dx, dy, dx + dot, dy + dot), fill=(34, 197, 94, 255))
    return img

sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
base = make(256)
base.save(r'C:\DevLok\desktop\assets\app.ico', sizes=[(s, s) for s in sizes])
base.save(r'C:\DevLok\desktop\assets\app.png')
print('ok')