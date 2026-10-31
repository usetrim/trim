#!/bin/sh
# ScriptPath already resolves /usr/share/trim/optimizer.py on Linux packages.
# Bootstrap still needs a one-time local pip install of llmlingua (not done at package install).
echo "Trim package installed. Next (same order as Quickstart):"
echo "  1) trim compress --bootstrap   # green success banner; local llmlingua (not Trim cloud)"
echo "  2) trim login"
echo "  3) trim start   # or: trim autostart enable"
echo "  optimizer.py is at /usr/share/trim/optimizer.py beside the CLI."
