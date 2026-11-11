# Clipboard dependency

Vendored from golang.design/x/clipboard v0.11.0, commit
`d95c7941706e35d9bf1d7a7560b0d8976c034712`, under the accompanying MIT license.
This is an external dependency module, not a second MECA product module.

MECA uses the read-only `ReadImage` extension: one request owns its connection,
cancellation and byte budget. It returns original raster bytes or one local file
URL; the MECA image codec owns pixel validation and normalization. The upstream
read/write/watch API is retained for provenance but is not used by MECA.

Only the macOS/Linux source subset is distributed; mobile, browser, BSD and native
Windows adapters are omitted. WSL uses the Windows host bridge in MECA.

Local changes add bounded, cancellable reads; X11 INCR reception; bounded Wayland
metadata and descriptor cleanup; and lazy macOS framework initialization. The
X11 authority reader rejects blocking nonregular files and oversized credentials.
The modified macOS adapter is split into format/lifecycle, publishing and native value
access files; NSString UTF-8 conversion uses purego's typed bridge. All changed or
added Go files follow the 350-line rule. Unchanged upstream files retain their
original organization and copyright headers so
dependency updates remain reviewable; they are third-party source, not authored
product code. Do not use the upstream unbounded image APIs from MECA.

Updates must preserve the bounded extension and these source splits, review the
upstream diff and license, run both modules' tests, and verify private macOS
pasteboard, Linux Wayland descriptor transfers and large X11 INCR transfers.
WSL's host interop must additionally be checked on Windows; Linux mock bridges do
not establish native Windows behavior.
