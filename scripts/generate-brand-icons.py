#!/usr/bin/env python3
"""Rasterize the owner-supplied Prosody SVG into native launcher assets.

Requires the existing rsvg-convert command. No network or image synthesis.
The opaque square artwork leaves corner masking to the platform.
"""
import json
import subprocess
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'mobile/assets/brand/prosody-mark.svg'


def render(destination, size, dark=False):
    """Render the original vector path with brand clear space and opaque fill."""
    path = ET.fromstring(SOURCE.read_text())[0].attrib['d']
    foreground, background = ('#FFFFFF', '#0A0A0A') if dark else ('#0A0A0A', '#FFFFFF')
    stroke = 14 if size < 32 else 12
    svg = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" '
           f'viewBox="0 0 100 100"><rect width="100" height="100" fill="{background}"/>'
           f'<g transform="translate(11 11) scale(.78)"><path d="{path}" fill="none" '
           f'stroke="{foreground}" stroke-width="{stroke}" stroke-linecap="round" '
           'stroke-linejoin="round"/></g></svg>')
    subprocess.run(['rsvg-convert', '-o', str(destination)], input=svg.encode(), check=True)


def main():
    """Regenerate every declared iOS icon and existing Android launcher density."""
    directory = ROOT / 'mobile/ios/Runner/Assets.xcassets/AppIcon.appiconset'
    catalog = json.loads((directory / 'Contents.json').read_text())
    for item in catalog['images']:
        size = int(float(item['size'].split('x')[0]) * float(item.get('scale', '1x')[:-1]))
        render(directory / item['filename'], size, bool(item.get('appearances')))
    for density, size in [('mdpi', 48), ('hdpi', 72), ('xhdpi', 96), ('xxhdpi', 144), ('xxxhdpi', 192)]:
        render(ROOT / f'mobile/android/app/src/main/res/mipmap-{density}/ic_launcher.png', size)


if __name__ == '__main__':
    main()
