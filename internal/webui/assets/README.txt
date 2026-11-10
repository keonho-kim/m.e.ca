MECA full embeds reproducibly built assets from webui/.
Run sh scripts/build-ui.sh from the repository root before building full.
The build packages hashed gzip assets, the manifest and browser license notices.
Generated output is ignored by Git. This placeholder supports source checkout.
The UI server refuses missing or incomplete asset manifests.
The tui build excludes the WebUI adapter and assets and requires no Bun.
