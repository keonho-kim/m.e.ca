# Example assets and distribution

The six fictional application identities and their source are original MECA studies. Sample names, dates, quantities and spaces are illustrative. Each guide explains design decisions; source HTML demonstrates the corresponding local behavior.

## Room images

The shared assets include two generated illustrative photographs:

- `studio-daylight.jpg`: a fictional creative workspace with windows, oak tables, cream seating and plants.
- `studio-warm.jpg`: a fictional meeting room with a walnut table, charcoal chairs and warm afternoon light.

Both originals are 1448×1086 JPEGs under 1MiB. Their matching 40px preview images support Blur-up. Preserve the correspondence between a small preview and its full image, and label the spaces illustrative. A real application should use media that accurately represents its own content.

## Icons

The local sprite contains 39 unmodified paths from [Lucide revision 53fab88ff9d1b48f4704b5b5b560d6b4d560bf4c](https://github.com/lucide-icons/lucide/tree/53fab88ff9d1b48f4704b5b5b560d6b4d560bf4c). `assets/examples/shared/icons-LICENSE.txt` contains the ISC and Feather-derived MIT notices. Preserve those notices when redistributing the sprite.

The examples use local symbols without an icon package or CDN. An application with an established icon system should use that system.

## Screen previews

Ten optional 1440×900 JPEGs in `assets/examples/previews/` show booking compositions, a spatial canvas, database inspection, approval, research, document, slides and the Archive collection. Each file is below 512KiB. They are captures of the bundled HTML, useful for comparing layout and hierarchy; source HTML defines navigation and interaction.

Approval, database, research and office examples are design shells. Archive and booking demonstrate bounded local state, and the canvas and motion examples have page-local behavior. These fixtures do not supply a production backend, database connection or model service.

## Reuse and notices

The collection's palettes, framework-free code and DESIGN/PRODUCT files describe these examples. The target project's requirements and design system determine its application architecture and visual identity.

The skill's NOTICE.md and focused source guides record external references and applicable licenses. Preserve attribution for copied licensed material. Bundled examples run with local HTML, CSS, Vanilla JS and media; no external font, image service or design plugin is required.

Markdown maps provide access to individual files. The three reading levels, documentation budgets and source-header conventions apply to this MECA bundle. Applications built with the skill follow their own project rules.
