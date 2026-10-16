"""
Generate multi-resolution Windows icon (logo/app.ico) from logo/01-as-apex.png.
Supported standard Windows resolutions: 16x16, 32x32, 48x48, 64x64, 128x128, 256x256
"""
import os
import sys
from PIL import Image

def generate_ico():
    src_png = os.path.join("logo", "01-as-apex.png")
    dst_ico = os.path.join("logo", "app.ico")
    if not os.path.exists(src_png):
        print(f"Error: source {src_png} does not exist", file=sys.stderr)
        sys.exit(1)
    img = Image.open(src_png)
    sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    img.save(dst_ico, format="ICO", sizes=sizes)
    print(f"Successfully generated {dst_ico} with {len(sizes)} resolutions: {sizes}")

if __name__ == "__main__":
    generate_ico()
