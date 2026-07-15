[CmdletBinding()]
param(
    [switch]$RebuildDocker,
    [string]$Branch = "main",
    [string]$PullRequest = "5062"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$logDirectory = Join-Path $repoRoot "logs"
$logFile = Join-Path $logDirectory "auto-update.log"
$mutex = [System.Threading.Mutex]::new($false, "Local\TonkicApiAutoUpdate")
$hasLock = $false

function Write-UpdateLog {
    param([string]$Message)

    $line = "{0} {1}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $Message
    Write-Host $line
    Add-Content -LiteralPath $logFile -Value $line -Encoding UTF8
}

function Invoke-Git {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)

    & git @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "git $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

try {
    New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
    $hasLock = $mutex.WaitOne(0)
    if (-not $hasLock) {
        Write-UpdateLog "Another update is already running; exiting."
        exit 0
    }

    Set-Location -LiteralPath $repoRoot
    if (-not (Test-Path -LiteralPath (Join-Path $repoRoot ".git"))) {
        throw "$repoRoot is not a Git repository"
    }

    $dirtyFiles = @(& git status --porcelain)
    if ($LASTEXITCODE -ne 0) {
        throw "Unable to inspect the working tree"
    }
    if ($dirtyFiles.Count -gt 0) {
        throw "Working tree is not clean. Commit or stash changes before automatic updating."
    }

    Write-UpdateLog "Fetching Tonkic/$Branch."
    Invoke-Git fetch --prune origin $Branch
    Invoke-Git checkout $Branch
    Invoke-Git merge --no-edit "origin/$Branch"

    Write-UpdateLog "Fetching QuantumNous/new-api pull request #$PullRequest."
    Invoke-Git fetch upstream "pull/$PullRequest/head:refs/remotes/upstream/pr-$PullRequest" --force
    Invoke-Git merge --no-edit "refs/remotes/upstream/pr-$PullRequest"

    if ($RebuildDocker) {
        if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
            throw "Docker is not installed or not available in PATH"
        }

        Write-UpdateLog "Building the merged source as calciumion/new-api:latest."
        & docker build --tag calciumion/new-api:latest .
        if ($LASTEXITCODE -ne 0) {
            throw "Docker image build failed with exit code $LASTEXITCODE"
        }

        & docker compose up -d --no-deps new-api
        if ($LASTEXITCODE -ne 0) {
            throw "Docker Compose restart failed with exit code $LASTEXITCODE"
        }
    }

    Write-UpdateLog "Update completed at commit $(& git rev-parse --short HEAD)."
}
catch {
    Write-UpdateLog "Update failed: $($_.Exception.Message)"
    exit 1
}
finally {
    if ($hasLock) {
        $mutex.ReleaseMutex()
    }
    $mutex.Dispose()
}
