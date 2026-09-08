"""
ApiClient Logo Generator (v4)

设计说明：
- 橙色系渐变圆角方形背景（浅橙 #FFB25B -> 深橙 #F26A1B，带轻微对角过渡）
- 中央白色"发送请求"箭头，造型简洁、小尺寸下依然清晰可辨
- 4x 超采样 + LANCZOS 缩放，保证边缘平滑

产出：
- build/appicon.png            512x512（wails 构建用）
- build/windows/icon.ico       多尺寸 Windows 图标
- frontend/src/assets/images/logo-16.png / logo-32.png / logo-universal.png
"""
import os
from PIL import Image, ImageDraw

SS = 4  # 超采样倍数

# 渐变色（左上浅 -> 右下深），橙色系
COLOR_TOP = (255, 196, 113)     # #FFC471  暖橘黄
COLOR_BOTTOM = (230, 90, 22)    # #E65A16  深橙
GLYPH_COLOR = (255, 255, 255)   # 白色箭头


def rounded_rect(draw, box, radius, fill):
    """绘制圆角矩形（自动处理超采样下的半径）"""
    x1, y1, x2, y2 = box
    draw.rounded_rectangle([x1, y1, x2, y2], radius=radius, fill=fill)


def build_gradient(w, h):
    """对角方向渐变背景"""
    img = Image.new('RGB', (w, h))
    px = img.load()
    for y in range(h):
        ty = y / max(h - 1, 1)
        for x in range(w):
            tx = x / max(w - 1, 1)
            t = min(max(0.55 * ty + 0.45 * tx, 0.0), 1.0)
            px[x, y] = (
                int(COLOR_TOP[0] + (COLOR_BOTTOM[0] - COLOR_TOP[0]) * t),
                int(COLOR_TOP[1] + (COLOR_BOTTOM[1] - COLOR_TOP[1]) * t),
                int(COLOR_TOP[2] + (COLOR_BOTTOM[2] - COLOR_TOP[2]) * t),
            )
    return img


def build_shape_mask(size, radius_ratio):
    """圆角方形遮罩（超采样后降采样，得到平滑边缘）"""
    big = size * SS
    mask = Image.new('L', (big, big), 0)
    draw = ImageDraw.Draw(mask)
    rounded_rect(draw, [0, 0, big - 1, big - 1], int(size * radius_ratio) * SS, 255)
    return mask.resize((size, size), Image.LANCZOS)


def build_glyph(size):
    """白色向右箭头（带圆角箭身 + 三角箭头）"""
    big = size * SS
    img = Image.new('RGBA', (big, big), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    s = big
    cx = cy = s / 2

    length = s * 0.54        # 箭头总长
    shaft_h = s * 0.115      # 箭身厚度
    head_w = s * 0.215       # 箭头三角宽度
    head_half = s * 0.185    # 箭头三角半高

    x_start = cx - length / 2
    x_end = cx + length / 2
    head_base = x_end - head_w

    # 箭身：圆角矩形，右端略微延伸进三角内避免出现缝隙
    rounded_rect(
        draw,
        [x_start, cy - shaft_h / 2, head_base + shaft_h * 0.6, cy + shaft_h / 2],
        shaft_h / 2,
        GLYPH_COLOR + (255,),
    )
    # 箭头：三角形
    draw.polygon(
        [(head_base, cy - head_half), (x_end, cy), (head_base, cy + head_half)],
        fill=GLYPH_COLOR + (255,),
    )

    return img.resize((size, size), Image.LANCZOS)


def generate_logo(size=512, radius_ratio=0.22):
    """生成指定尺寸的 Logo（RGBA）"""
    gradient = build_gradient(size, size)
    mask = build_shape_mask(size, radius_ratio)

    canvas = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    canvas.paste(gradient, (0, 0), mask)

    glyph = build_glyph(size)
    return Image.alpha_composite(canvas, glyph)


def save_ico(logo, path, sizes=((16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256))):
    """保存多尺寸 ICO"""
    images = [logo.resize(s, Image.LANCZOS) for s in sizes]
    images[-1].save(path, format='ICO', sizes=list(sizes), append_images=images[:-1])


if __name__ == '__main__':
    output_dir = os.path.dirname(os.path.abspath(__file__))
    frontend_dir = os.path.join(output_dir, '..', 'frontend', 'src', 'assets', 'images')
    os.makedirs(frontend_dir, exist_ok=True)

    logo = generate_logo(512)

    logo.save(os.path.join(output_dir, 'appicon.png'), 'PNG')
    print('Saved appicon.png (512x512)')

    for name, size in (('logo-16.png', 16), ('logo-32.png', 32), ('logo-universal.png', 128)):
        logo.resize((size, size), Image.LANCZOS).save(os.path.join(frontend_dir, name), 'PNG')
        print(f'Saved {name} ({size}x{size})')

    ico_path = os.path.join(output_dir, 'windows', 'icon.ico')
    save_ico(logo, ico_path)
    print('Saved windows/icon.ico (16/32/48/64/128/256)')

    print('\nLogo generation complete!')
