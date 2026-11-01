# Moonrider website branding

Observed on [Moonrider's official homepage](https://www.moonrider.ai/) on 1 November 2026. These are website-derived references, not a separately published brand standards manual.

## Applied identity

| Element | Source observation | App use |
| --- | --- | --- |
| Primary orange | `#E7920D`, including the original SVG logo | Live indicator, navigation selection, metric accents |
| Dark neutral | `#212121` | Headings and primary content |
| White | `#FFFFFF` | Cards and navigation |
| Secondary text | `#5D5D5D` | Labels and supporting copy |
| Body type | Poppins | Locally bundled weights 400, 500, 600, 700 |
| Heading type | Orbitron | Locally bundled heading font |
| Logo | Original orange-and-black SVG wordmark and symbol | Header, preserving its original artwork and colors |
| Website icon | Original 32px PNG favicon | Browser favicon and web app icon |

The logo sits on a white header so its black letterforms remain legible. The app's `#F7F7F7` page background and `#FFF4E4` orange-tinted status surface are neutral/tint adaptations. Alert red (`#B42318`) and success green (`#267445`) are functional status colors, not claimed brand standards. Orange remains the dominant brand accent. The blue `#0050BD` seen in generic website focus styles was not adopted as a brand color.

The homepage headline is “Revolutionizing the Future of Farm Mechanization,” with “ev” emphasized in orange. No separate official tagline was confirmed, so the previous invented header tagline was removed. Screen titles and supporting product copy are app-specific.

## Downloaded assets and provenance

All app assets are local so the logo and typography do not depend on runtime network requests.

- [Original SVG logo](https://cdn.prod.website-files.com/6a0a9300ec4156fb48ce3540/6a0ab6fe44b73e075eb51258_logo.svg): `assets/brand/moonrider-logo.svg`
- [Original website favicon](https://cdn.prod.website-files.com/6a0a9300ec4156fb48ce3540/6a30d2400a604dc207d6ca27_Icon_Light%20Mode_bg.png): `assets/brand/moonrider-icon.png` and `web/moonrider-icon.png`
- [Homepage stylesheet used to verify the palette and font families](https://cdn.prod.website-files.com/6a0a9300ec4156fb48ce3540/css/moonrider-fef676.webflow.shared.a5f63f590.css)
- [Poppins font files and license](https://github.com/google/fonts/tree/main/ofl/poppins): `assets/fonts/Poppins-*.ttf` and `Poppins-OFL.txt`
- [Orbitron font file and license](https://github.com/google/fonts/tree/main/ofl/orbitron): `assets/fonts/Orbitron.ttf` and `Orbitron-OFL.txt`

The favicon source is 32px; no larger source is claimed. Native Android/iOS launcher icons are unchanged by this theme update. This remains an independent portfolio prototype; use of existing branding does not imply company endorsement or transfer of rights.

## Implementation

`lib/ui/brand.dart` centralizes the palette and Material theme. `pubspec.yaml` declares the bundled fonts and brand assets. The header uses `flutter_svg` to render the original logo with an accessible label. Screens and status components use the shared tokens. The README screenshots are refreshed after visual verification of the branded app.
