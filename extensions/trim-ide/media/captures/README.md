# Raw captures (not published)

Put live screenshots here before processing:

- `setup.png` → becomes `../screenshot-setup.png`
- `status.png` → becomes `../screenshot-status.png`

```powershell
python scripts/_process_ext_screenshots.py
```

Raw files in this folder are gitignored; only the processed `screenshot-*.png` are shipped in the VSIX.
