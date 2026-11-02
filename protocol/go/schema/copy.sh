#!/bin/sh
# Copies schemas/ from the repository root into files/.
rm -rf files
mkdir -p files/payloads
cp ../../../schemas/*.schema.json files/
cp ../../../schemas/payloads/*.schema.json files/payloads/
