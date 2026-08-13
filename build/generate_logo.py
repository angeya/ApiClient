"""
ApiClient Logo Generator
生成圆角风格的 API 客户端 Logo，包含多种尺寸
"""
from PIL import Image, ImageDraw, ImageFont
import math
import os

def rounded_rectangle(draw, bbox, radius, fill):
    """绘制圆角矩形"""
    x1, y1, x2, y2 = bbox
    draw.rectangle([x1 + radius, y1, x2 - radius, y2], fill=fill)
    draw.rectangle([x1, y1 + radius, x2, y2 - radius], fill=fill)
    draw.pieslice([x1, y1, x1 + 2 * radius, y1 + 2 * radius], 180, 270, fill=fill)
    draw.pieslice([x2 - 2 * radius, y1, x2, y1 + 2 * radius], 270, 360, fill=fill)
    draw.pieslice([x1, y2 - 2 * radius, x1 + 2 * radius, y2], 90, 180, fill=fill)
    draw.pieslice([x2 - 2 * radius, y2 - 2 * radius, x2, y2], 0, 90, fill=fill)


def draw_arrow(draw, cx, cy, size, fill):
    """绘制向右的发送箭头"""
    w = size
    h = int(size * 0.7)
    # 箭头（三角形）
    arrow_w = int(w * 0.35)
    shaft_w = w - arrow_w
    shaft_h = int(h * 0.28)
    # 箭身
    x_start = cx - w // 2
    y_center = cy
    shaft_x1 = x_start
    shaft_x2 = x_start + shaft_w
    draw.rectangle([shaft_x1, y_center - shaft_h // 2, shaft_x2, y_center + shaft_h // 2], fill=fill)
    # 箭头三角
    tip_x = x_start + w
    arrow_half_h = int(h * 0.5)
    draw.polygon([
        (shaft_x2, y_center - arrow_half_h),
        (tip_x, y_center),
        (shaft_x2, y_center + arrow_half_h),
    ], fill=fill)


def draw_brackets(draw, cx, cy, size, fill):
    """绘制花括号 { }"""
    h = size
    w = int(size * 0.25)
    gap = int(size * 0.55)
    thickness = max(int(size * 0.1), 2)

    # 左花括号 {
    lx = cx - gap // 2
    for t in range(thickness):
        # 上半
        draw.arc([lx - w + t, cy - h // 2, lx + t, cy - t], 90, 180, fill=fill)
        # 下半
        draw.arc([lx - w + t, cy + t, lx + t, cy + h // 2], 180, 270, fill=fill)
        # 中间连接
        draw.rectangle([lx - w + t, cy - 1, lx + t, cy + 1], fill=fill)

    # 右花括号 }
    rx = cx + gap // 2
    for t in range(thickness):
        draw.arc([rx - t, cy - h // 2, rx + w - t, cy - t], 0, 90, fill=fill)
        draw.arc([rx - t, cy + t, rx + w - t, cy + h // 2], 270, 360, fill=fill)
        draw.rectangle([rx - t, cy - 1, rx + w - t, cy + 1], fill=fill)


def generate_logo(size=512, bg_color=(59, 130, 246), icon_color=(255, 255, 255)):
    """生成指定尺寸的 Logo"""
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # 圆角背景
    padding = int(size * 0.04)
    radius = int(size * 0.18)
    rounded_rectangle(draw, [padding, padding, size - padding, size - padding],
                       radius, fill=bg_color + (255,))

    # 绘制内容区域
    content_area = size - 2 * padding
    cx, cy = size // 2, size // 2

    # 上方：花括号 { }
    bracket_size = int(content_area * 0.32)
    bracket_y = cy - int(content_area * 0.12)
    draw_brackets(draw, cx, bracket_y, bracket_size, icon_color + (255,))

    # 下方：发送箭头
    arrow_size = int(content_area * 0.28)
    arrow_y = cy + int(content_area * 0.2)
    draw_arrow(draw, cx, arrow_y, arrow_size, icon_color + (255,))

    return img


def generate_logo_v2(size=512, bg_color=(59, 130, 246), icon_color=(255, 255, 255)):
    """
    生成 v2 版本 Logo：
    圆角方形背景 + 中间一个向右的请求箭头穿过圆环（代表网络/API循环）
    """
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    padding = int(size * 0.04)
    radius = int(size * 0.18)
    rounded_rectangle(draw, [padding, padding, size - padding, size - padding],
                       radius, fill=bg_color + (255,))

    cx, cy = size // 2, size // 2
    content_size = size - 2 * padding

    # 圆环（代表网络/API）
    ring_r = int(content_size * 0.28)
    ring_thickness = max(int(content_size * 0.055), 3)
    for t in range(ring_thickness):
        r = ring_r - t // 2
        draw.ellipse([cx - r, cy - r, cx + r, cy + r], outline=icon_color + (255,))

    # 向右箭头穿过圆环
    arrow_len = int(content_size * 0.55)
    arrow_h = int(content_size * 0.13)
    arrow_head_w = int(content_size * 0.12)
    ax1 = cx - arrow_len // 2
    ax2 = cx + arrow_len // 2 - arrow_head_w
    draw.rectangle([ax1, cy - arrow_h // 2, ax2, cy + arrow_h // 2], fill=icon_color + (255,))
    # 箭头
    draw.polygon([
        (ax2, cy - int(content_size * 0.15)),
        (cx + arrow_len // 2, cy),
        (ax2, cy + int(content_size * 0.15)),
    ], fill=icon_color + (255,))

    return img


def generate_logo_v3(size=512):
    """
    v3: 渐变蓝色圆角方形 + 白色 API 图标
    图标: 上方 "API" 文字, 下方一个右箭头(代表请求发送)
    简洁专业，辨识度高
    """
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    padding = int(size * 0.02)
    radius = int(size * 0.20)

    # 渐变背景 - 从深蓝到亮蓝
    for y in range(size):
        ratio = y / size
        r = int(30 + (59 - 30) * ratio)
        g = int(64 + (130 - 64) * ratio)
        b = int(175 + (246 - 175) * ratio)
        for x in range(size):
            # 只在圆角矩形内绘制
            in_rect = True
            # 检查四个角
            corners = [
                (padding + radius, padding + radius),
                (size - padding - radius, padding + radius),
                (padding + radius, size - padding - radius),
                (size - padding - radius, size - padding - radius),
            ]
            if y < padding or y >= size - padding or x < padding or x >= size - padding:
                in_rect = False
            elif x < padding + radius and y < padding + radius:
                dx = x - (padding + radius)
                dy = y - (padding + radius)
                if dx * dx + dy * dy > radius * radius:
                    in_rect = False
            elif x > size - padding - radius and y < padding + radius:
                dx = x - (size - padding - radius)
                dy = y - (padding + radius)
                if dx * dx + dy * dy > radius * radius:
                    in_rect = False
            elif x < padding + radius and y > size - padding - radius:
                dx = x - (padding + radius)
                dy = y - (size - padding - radius)
                if dx * dx + dy * dy > radius * radius:
                    in_rect = False
            elif x > size - padding - radius and y > size - padding - radius:
                dx = x - (size - padding - radius)
                dy = y - (size - padding - radius)
                if dx * dx + dy * dy > radius * radius:
                    in_rect = False

            if in_rect:
                img.putpixel((x, y), (r, g, b, 255))

    draw = ImageDraw.Draw(img)
    cx, cy = size // 2, size // 2
    content_size = size - 2 * padding

    # 绘制 API 文字
    try:
        font_size = int(content_size * 0.28)
        font = ImageFont.truetype("arial.ttf", font_size)
    except:
        try:
            font = ImageFont.truetype("C:/Windows/Fonts/arial.ttf", int(content_size * 0.28))
        except:
            font = ImageFont.load_default()

    text = "API"
    bbox = draw.textbbox((0, 0), text, font=font)
    tw = bbox[2] - bbox[0]
    th = bbox[3] - bbox[1]
    tx = cx - tw // 2
    ty = cy - int(content_size * 0.22) - th // 2
    draw.text((tx, ty), text, fill=(255, 255, 255, 255), font=font)

    # 发送箭头
    arrow_y = cy + int(content_size * 0.15)
    arrow_len = int(content_size * 0.40)
    shaft_h = max(int(content_size * 0.055), 3)
    head_h = int(content_size * 0.14)

    ax1 = cx - arrow_len // 2
    ax2 = cx + arrow_len // 2
    head_start = ax2 - int(content_size * 0.10)

    draw.rectangle([ax1, arrow_y - shaft_h // 2, head_start, arrow_y + shaft_h // 2],
                    fill=(255, 255, 255, 255))
    draw.polygon([
        (head_start, arrow_y - head_h),
        (ax2, arrow_y),
        (head_start, arrow_y + head_h),
    ], fill=(255, 255, 255, 255))

    return img


if __name__ == '__main__':
    output_dir = os.path.dirname(os.path.abspath(__file__))
    icon_dir = os.path.join(output_dir, 'windows')

    # 生成 v3 Logo（最佳版本）
    logo = generate_logo_v3(512)

    # 保存 512x512 PNG (用于 build/appicon.png)
    logo.save(os.path.join(output_dir, 'appicon.png'), 'PNG')
    print("Saved appicon.png (512x512)")

    # 保存 32x32 用于前端标题栏
    small = logo.resize((32, 32), Image.LANCZOS)
    frontend_dir = os.path.join(output_dir, '..', 'frontend', 'src', 'assets', 'images')
    os.makedirs(frontend_dir, exist_ok=True)
    small.save(os.path.join(frontend_dir, 'logo-32.png'), 'PNG')
    print("Saved logo-32.png (32x32)")

    # 保存 16x16 用于标题栏
    tiny = logo.resize((16, 16), Image.LANCZOS)
    tiny.save(os.path.join(frontend_dir, 'logo-16.png'), 'PNG')
    print("Saved logo-16.png (16x16)")

    # 保存 universal logo
    universal = logo.resize((128, 128), Image.LANCZOS)
    universal.save(os.path.join(frontend_dir, 'logo-universal.png'), 'PNG')
    print("Saved logo-universal.png (128x128)")

    # 生成 ICO 文件 (Windows 任务栏图标)
    ico_sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    ico_images = []
    for s in ico_sizes:
        resized = logo.resize(s, Image.LANCZOS)
        # ICO 需要 RGBA 转为 RGB 的白色背景，避免透明问题
        bg = Image.new('RGBA', s, (255, 255, 255, 0))
        composite = Image.alpha_composite(bg, resized)
        ico_images.append(composite)
    ico_path = os.path.join(icon_dir, 'icon.ico')
    ico_images[-1].save(ico_path, format='ICO', sizes=ico_sizes,
                         append_images=ico_images[:-1])
    print(f"Saved icon.ico with sizes: {[s for s in ico_sizes]}")

    print("\nLogo generation complete!")
