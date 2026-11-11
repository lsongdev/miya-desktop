# Native application assets

`appicon.png` is the source icon for the native interface, tray and updater.
MyGo generates platform icons, bundles, manifests, version resources and
installers from `../mygo.json`.

`make build` writes packages to `../bin/<os>-<arch>/`.
The entire application is Go; there is no frontend asset build.
See the root README for packaging and update-signing instructions.
