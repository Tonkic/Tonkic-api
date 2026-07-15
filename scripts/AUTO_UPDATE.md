# Automatic updates

Run an update manually from PowerShell:

```powershell
& D:\Tonkic-api\scripts\auto-update.ps1
```

The script fetches and merges `origin/main`, then fetches and merges
`QuantumNous/new-api` pull request `#5062`. It refuses to run with uncommitted
changes and records results in `logs/auto-update.log`.

To rebuild the merged source and restart the `new-api` Docker Compose service:

```powershell
& D:\Tonkic-api\scripts\auto-update.ps1 -RebuildDocker
```

Install a daily Windows scheduled task (default: 04:00):

```powershell
& D:\Tonkic-api\scripts\install-auto-update-task.ps1
```

Choose another time and enable automatic Docker deployment:

```powershell
& D:\Tonkic-api\scripts\install-auto-update-task.ps1 -DailyAt "03:30" -RebuildDocker
```

Installing the task requires an account allowed to register Windows scheduled
tasks. The task is intentionally not installed by the repository setup itself.
