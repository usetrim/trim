#!/bin/sh
# ScriptPath already resolves /usr/share/trim/optimizer.py on Linux packages.
# Bootstrap still needs a one-time local pip install of llmlingua (not done at package install).
echo "Deep Mode: trim compress --bootstrap"
echo "  (optimizer.py is at /usr/share/trim/optimizer.py; pip installs Microsoft LLMLingua on this machine only)"
