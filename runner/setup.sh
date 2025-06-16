#!/bin/bash
set -e

echo "[🔥] Booting setup-runner.sh"

# === CONFIG ===
REPO_OWNER="2pai"
REPO_NAME="mvm-github-runner"
RUNNER_LABEL="runner-firecracker"
 
METADATA=$(curl -s -H "Accept: application/json" http://169.254.169.254/)
PAT_TOKEN=$(echo "$METADATA" | jq -r .RUNNER_PAT)
RUNNER_NAME=$(echo "$METADATA" | jq -r .RUNNER_NAME)

PAT_TOKEN=""
RUNNER_HOME="/home/runner"
WORK_DIR="${RUNNER_HOME}/_work"

mkdir -p /proc /sys /sys/fs/cgroup /dev /dev/pts /tmp
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t tmpfs tmpfs /sys/fs/cgroup
mount -o bind /dev /dev
mount -o bind /dev/pts /dev/pts

# === Setup PATH + DNS ===
export PATH=/usr/sbin:/usr/bin:/sbin:/bin

echo "[*] Setting up resolv.conf..."
echo "nameserver 8.8.8.8" > /etc/resolv.conf

ping -c2 1.1.1.1

# === GET REGISTRATION TOKEN ===
echo "[*] Fetching runner registration token... for ${REPO_OWNER}/${REPO_NAME} (${RUNNER_NAME})"

RUNNER_TOKEN=$(curl -s -X POST \
  -H "Authorization: token ${PAT_TOKEN}" \
  "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/actions/runners/registration-token" | jq -r .token)

if [[ -z "$RUNNER_TOKEN" || "$RUNNER_TOKEN" == "null" ]]; then
  echo "[!] Failed to fetch runner token. Check PAT or repo permissions. ($RUNNER_TOKEN)"
  exit 1
fi

# === RUN RUNNER ===
cd "$RUNNER_HOME"
chown -R runner:runner "$RUNNER_HOME"

if [ -f ".runner" ]; then
  echo "[*] Cleaning previous runner config"
  ./config.sh remove --token "$RUNNER_TOKEN"
fi

su - runner -c "./config.sh --unattended --ephemeral \
  --url https://github.com/${REPO_OWNER}/${REPO_NAME} \
  --token ${RUNNER_TOKEN} --labels ${RUNNER_LABEL} --name ${RUNNER_NAME}"

echo "[*] Running GitHub runner..."
su - runner -c "./run.sh" && halt -f
