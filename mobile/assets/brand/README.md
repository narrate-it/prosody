# Prosody brand

Owner-supplied `prosody-mark.svg` is the source of the launcher icons and the
matching in-app vector path. The mark is one continuous rounded stroke.

Palette: Ink `#0A0A0A`, Paper `#FFFFFF`, Fog `#F5F6F8`, Graphite `#5C5C62`.
Signal `#FF4F1A` is reserved for unread indicators; it is not an action color.
Dark surfaces use neutral elevations and lighter secondary text for contrast.
Errors retain a distinct accessible semantic red. Existing typography is unchanged.

Regenerate committed native icons with `python3 scripts/generate-brand-icons.py`
from the repository root (requires `rsvg-convert`). Icons are opaque squares;
iOS supplies the corner mask. Light uses Ink on Paper; dark/tinted use Paper on
Ink. The mark occupies approximately 64% of the tile, centered in its viewBox.
Below 32 pixels the stroke is slightly thicker, per the guide.

This source change does not modify the already uploaded build 41. A subsequent
reviewed, signed upload is required for the new branding to appear in TestFlight.
