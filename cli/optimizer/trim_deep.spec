# PyInstaller spec for optional zero-deps Deep Mode binary (trim-deep).
# Build on a machine with Python 3.10+ and: pip install pyinstaller llmlingua
#   pyinstaller cli/optimizer/trim_deep.spec
# Output: dist/trim-deep (or trim-deep.exe). Pack next to the trim CLI binary.
# Never deploy this model bundle on Render free tier.

block_cipher = None

a = Analysis(
    ["optimizer.py"],
    pathex=[],
    binaries=[],
    datas=[],
    hiddenimports=[
        "llmlingua",
        "llmlingua.prompt_compressor",
        "transformers",
        "torch",
        "tokenizers",
    ],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name="trim-deep",
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)
