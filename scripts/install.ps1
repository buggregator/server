# Buggregator installer for Windows (PowerShell 5.1+).
#
#   irm https://raw.githubusercontent.com/buggregator/server/master/scripts/install.ps1 | iex
#
# Downloads the binary for this CPU from a GitHub release, checks it against
# the release's checksums.txt, and puts it in %LOCALAPPDATA%\Programs\buggregator.
# With OpenSSL 3 on PATH (e.g. from Git for Windows) it also checks
# checksums.txt.sig, the release's ed25519 signature; Windows has no ed25519 of its own.
#
# Environment:
#   BUGGREGATOR_VERSION      release tag, e.g. v2.6.0 (default: latest)
#   BUGGREGATOR_INSTALL_DIR  where to put the binary
#   BUGGREGATOR_BASE_URL     download base URL instead of GitHub (mirrors, tests)
#   BUGGREGATOR_NO_VERIFY=1  skip the checksum (only for releases without checksums.txt)
#   BUGGREGATOR_NO_SIGNATURE=1  install without checksums.txt.sig (releases from
#                            v2.11.0 on are signed; a missing signature stops the install)

# Runs in its own scope: with `irm | iex` nothing (preferences, variables) is left
# behind in the user's session.
& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'  # Invoke-WebRequest is very slow with the progress bar

    $repo = 'buggregator/server'
    $version = if ($env:BUGGREGATOR_VERSION) { $env:BUGGREGATOR_VERSION } else { 'latest' }
    $installDir = if ($env:BUGGREGATOR_INSTALL_DIR) { $env:BUGGREGATOR_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\buggregator' }

    # A 32-bit PowerShell on a 64-bit Windows reports x86 here; the real CPU is in PROCESSOR_ARCHITEW6432.
    $cpu = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    switch ($cpu) {
        'AMD64' { $arch = 'amd64' }
        'ARM64' { $arch = 'arm64' }
        default { throw "buggregator install: unsupported CPU $cpu" }
    }
    $asset = "buggregator-windows-$arch.exe"

    if ($env:BUGGREGATOR_BASE_URL) { $base = $env:BUGGREGATOR_BASE_URL }
    elseif ($version -eq 'latest') { $base = "https://github.com/$repo/releases/latest/download" }
    else { $base = "https://github.com/$repo/releases/download/$version" }

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("buggregator-" + [System.Guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        Write-Host "Downloading $asset ($version)..."
        $file = Join-Path $tmp $asset
        try { Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $file }
        catch { throw "buggregator install: download failed: $base/$asset ($($_.Exception.Message))" }

        if ($env:BUGGREGATOR_NO_VERIFY -eq '1') {
            Write-Host 'Checksum NOT verified (BUGGREGATOR_NO_VERIFY=1).'
        } else {
            $sums = Join-Path $tmp 'checksums.txt'
            try { Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile $sums }
            catch { throw 'buggregator install: no checksums.txt in this release; set BUGGREGATOR_NO_VERIFY=1 to install without a check' }
            $want = $null
            foreach ($line in Get-Content $sums) {
                $parts = $line -split '\s+'
                if ($parts.Count -ge 2 -and $parts[1].TrimStart('*') -eq $asset) { $want = $parts[0].ToLower() }
            }

            # The ed25519 key that signs checksums.txt (releaseKey in internal/selfupdate/verify.go).
            $sig = Join-Path $tmp 'checksums.txt.sig'
            $hasSig = $true
            try { Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt.sig" -OutFile $sig } catch { $hasSig = $false }
            $openssl = Get-Command openssl -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
            $sslVersion = if ($openssl) { (& $openssl.Source version 2>$null) -join ' ' } else { '' }
            # Releases from v2.11.0 on (and so "latest") are signed; a version we cannot read counts as signed.
            $signed = $true
            if ($version -match '^v?(\d+)\.(\d+)') {
                $signed = ([int]$Matches[1] -gt 2) -or ([int]$Matches[1] -eq 2 -and [int]$Matches[2] -ge 11)
            }
            if (-not $hasSig) {
                # Someone who replaced the binary and checksums.txt would also drop the signature.
                if ($env:BUGGREGATOR_NO_SIGNATURE -eq '1') { Write-Host 'Signature NOT checked (BUGGREGATOR_NO_SIGNATURE=1).' }
                elseif ($signed) { throw "buggregator install: no checksums.txt.sig: $version should be signed. Set BUGGREGATOR_NO_SIGNATURE=1 only for a mirror you trust that has no signature" }
                else { Write-Host "Signature not checked: $version predates signed releases (v2.11.0)." }
            } elseif ($sslVersion -notmatch '^OpenSSL ([3-9]|\d{2,})\.') {
                Write-Host 'Signature not checked: needs OpenSSL 3 on PATH (e.g. from Git for Windows).'
            } else {
                $pem = Join-Path $tmp 'release-key.pem'
                Set-Content -Path $pem -Encoding Ascii -Value @(
                    '-----BEGIN PUBLIC KEY-----'
                    'MCowBQYDK2VwAyEAbFthyZgjx8k7TUu4ZekM3sa0zSxSRzOTaX2BfxaUH2I='
                    '-----END PUBLIC KEY-----')
                & $openssl.Source pkeyutl -verify -pubin -inkey $pem -rawin -in $sums -sigfile $sig *> $null
                if ($LASTEXITCODE -ne 0) { throw 'buggregator install: the signature of checksums.txt does not match the Buggregator release key: do not use this download' }
                Write-Host 'Signature OK.'
            }

            if (-not $want) { throw "buggregator install: $asset is not listed in checksums.txt" }
            $got = (Get-FileHash -Algorithm SHA256 -Path $file).Hash.ToLower()
            if ($got -ne $want) { throw "buggregator install: checksum mismatch for $asset (got $got, want $want)" }
            Write-Host 'Checksum OK.'
        }

        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
        $bin = Join-Path $installDir 'buggregator.exe'
        $q = $bin -replace "'", "''"  # for the printed commands
        Move-Item -Force -Path $file -Destination $bin
    } finally {
        Remove-Item -Recurse -Force -Path $tmp -ErrorAction SilentlyContinue
    }

    Write-Host ''
    Write-Host "Installed $bin"
    Write-Host ''
    Write-Host 'Start it (web UI at http://127.0.0.1:8000):'
    Write-Host "  & '$q'"
    Write-Host ''
    Write-Host 'Only on this machine (no other hosts on the network):'
    Write-Host "  `$env:HTTP_ADDR='127.0.0.1:8000'; `$env:SMTP_ADDR='127.0.0.1:1025'; `$env:MONOLOG_ADDR='127.0.0.1:9913'; `$env:PROXY_ADDR='127.0.0.1:8080'; & '$q'"
    Write-Host ''
    Write-Host 'With the MCP server for AI agents (HTTP, this machine only), then connect Claude Code:'
    Write-Host "  `$env:MCP_ENABLED='true'; `$env:MCP_TRANSPORT='http'; `$env:MCP_ADDR='127.0.0.1:8001'; & '$q'"
    Write-Host '  claude mcp add --transport http buggregator http://127.0.0.1:8001/mcp'
    Write-Host ''
    Write-Host 'VarDumper (port 9912) is not available on Windows.'
}
