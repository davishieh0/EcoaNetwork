#!/bin/sh
# Copies protocol/schemas/ into files/.
rm -rf files
mkdir -p files/payloads
cp ../../schemas/*.schema.json files/
cp ../../schemas/payloads/*.schema.json files/payloads/
