# Automatic updates on Ubuntu/Linux

The updater requires Bash, Git, and `flock` (provided by `util-linux` on
Ubuntu). Run it manually from the repository root:

```bash
./scripts/auto-update.sh
```

It fetches and merges `origin/main`, then fetches and merges pull request
`QuantumNous/new-api#5062`. It refuses to run with uncommitted changes, prevents
concurrent runs, aborts a conflicted merge, and writes to
`logs/auto-update.log`.

To rebuild the merged source and recreate the `new-api` Docker Compose service:

```bash
./scripts/auto-update.sh --rebuild-docker
```

## Install the systemd timer

Install a daily timer at 04:00:

```bash
sudo ./scripts/install-auto-update.sh
```

Choose another time and enable automatic Docker deployment:

```bash
sudo ./scripts/install-auto-update.sh --time 03:30 --rebuild-docker
```

The service runs as the user who invoked `sudo`. For Docker deployment, that
user must be allowed to access the Docker daemon (typically by membership in
the `docker` group). Re-login after adding group membership.

Useful commands:

```bash
systemctl status tonkic-api-auto-update.timer
systemctl list-timers tonkic-api-auto-update.timer
journalctl -u tonkic-api-auto-update.service
sudo systemctl start tonkic-api-auto-update.service
```

To remove the timer and service:

```bash
sudo systemctl disable --now tonkic-api-auto-update.timer
sudo rm /etc/systemd/system/tonkic-api-auto-update.timer
sudo rm /etc/systemd/system/tonkic-api-auto-update.service
sudo systemctl daemon-reload
```
