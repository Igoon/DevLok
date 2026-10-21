from PIL import Image, ImageDraw, ImageFont

def make(size=256):
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    r = int(size * 0.22)
    d.rounded_rectangle((0, 0, size - 1, size - 1), radius=r, fill=(11, 18, 32, 255), outline=(34, 211, 238, 255), width=max(2, size // 40))
    font = ImageFont.truetype('C:/Windows/Fonts/arialbd.ttf', int(size * 0.46))
    text = 'DL'
    box = d.textbbox((0, 0), text, font=font)
    tw, th = box[2] - box[0], box[3] - box[1]
    d.text(((size - tw) / 2 - box[0], (size - th) / 2 - box[1] + size * 0.03), text, font=font, fill=(248, 250, 252, 255))
    dot = max(5, size // 10)
    dx = size - dot - int(size * 0.10)
    dy = int(size * 0.10)
    d.ellipse((dx, dy, dx + dot, dy + dot), fill=(34, 197, 94, 255))
    return img

sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
base = make(256)
base.save(r'C:\DevLok\desktop\assets\app.ico', sizes=[(s, s) for s in sizes])
base.save(r'C:\DevLok\desktop\assets\app.png')
print('ok')