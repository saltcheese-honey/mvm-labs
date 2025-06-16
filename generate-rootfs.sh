#!/bin/bash
set -euo pipefail

IMAGE_TAG="github-runner-firecracker:latest"
OUTPUT_IMG="rootfs.ext4"
SIZE_MB=4096

sudo umount mnt 2>/dev/null || true
rm -rf mnt rootfs rootfs.tar "$OUTPUT_IMG"
mkdir -p rootfs mnt

echo "[*] Building Docker image..."
docker build -f Dockerfile-runner -t $IMAGE_TAG .

echo "[*] Exporting container..."
CID=$(docker create $IMAGE_TAG)
docker export "$CID" -o rootfs.tar
docker rm "$CID"

echo "[*] Extracting filesystem..."
tar -xf rootfs.tar -C rootfs

echo "[*] Creating ext4 image..."
dd if=/dev/zero of=$OUTPUT_IMG bs=1M count=$SIZE_MB
mkfs.ext4 $OUTPUT_IMG

echo "[*] Copying into ext4 image..."
sudo mount -o loop $OUTPUT_IMG mnt
sudo cp -a rootfs/* mnt/
sudo umount mnt

echo "[✓] rootfs.ext4 ready!"
