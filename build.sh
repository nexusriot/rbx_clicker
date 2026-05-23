#!/usr/bin/env bash
set -e

OUTDIR="dist"
mkdir -p "$OUTDIR"

echo "Building ckicker.exe..."
GOOS=windows GOARCH=amd64 go build -o "$OUTDIR/ckicker.exe" ckicker.go

echo "Building clicker3.exe..."
GOOS=windows GOARCH=amd64 go build -o "$OUTDIR/clicker3.exe" clicker3.go

echo "Done. Binaries in $OUTDIR/"
ls -lh "$OUTDIR/"
