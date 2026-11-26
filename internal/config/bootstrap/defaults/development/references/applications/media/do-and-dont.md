# Media Do & Don’t

| Decision | Do | Don’t |
| --- | --- | --- |
| Playback readiness | Use a playable rendition's state and expose meaningful processing status. | Promise playback or quality from upload progress alone. |
| Rendition lifetime | Bind outputs, captions and previews to their source revision. | Attach late processing results from a previous upload to its replacement. |
| Visibility | Apply current audience policy to playable and derived resources. | Protect the page while serving its manifest or thumbnails outside the access contract. |
| Resume | Preserve a useful, valid position for the specific version. | Count buffering or seeking as watched time without a defined rule. |
| Retry | Retry the relevant upstream or derived phase and retain usable outputs where safe. | Transcode every variant again to retry one failed caption or quality. |
| Control | Keep playback, pause, seek and relevant captions operable in actual states. | Use a decorative playback state to hide a stalled or unsupported stream. |
