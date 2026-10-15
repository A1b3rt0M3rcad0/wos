$ErrorActionPreference = 'Stop'
# Opt-in Windows fixture: create and fill only a new, labelled, disposable VHD.
# Never select a physical disk or format an existing volume.
$fixtureId = [guid]::NewGuid().ToString('N').Substring(0, 12)
$fixtureRoot = Join-Path ([IO.Path]::GetTempPath()) "wos-full-disk-$fixtureId"
New-Item -ItemType Directory -Path $fixtureRoot | Out-Null
$vhd = Join-Path $fixtureRoot 'fixture.vhd'
$diskScript = Join-Path $fixtureRoot 'diskpart.txt'
$testBinary = Join-Path $fixtureRoot 'wos-tests.exe'
$label = "WOS-$fixtureId"
$drive = @('Z', 'Y', 'X', 'W', 'V', 'U', 'T', 'S', 'R') | Where-Object { -not (Test-Path "${_}:\") } | Select-Object -First 1
if (-not $drive) { throw 'No unused drive letter for the isolated VHD fixture' }
$previousFixture = $env:WOS_TEST_FULL_DISK_DIRECTORY
try {
    go test -c -o $testBinary ./packages/wos-cli
    if ($LASTEXITCODE -ne 0) { throw 'Could not compile native Windows CLI tests' }
    @"
create vdisk file="$vhd" maximum=32 type=expandable
select vdisk file="$vhd"
attach vdisk
"@ | Set-Content -Encoding ascii $diskScript
    diskpart.exe /s $diskScript
    if ($LASTEXITCODE -ne 0) { throw 'Could not provision isolated capacity fixture' }
    $ownedDisk = Get-DiskImage -ImagePath $vhd | Get-Disk
    if (-not $ownedDisk -or $ownedDisk.IsBoot -or $ownedDisk.IsSystem -or $ownedDisk.Size -ge 64MB -or $ownedDisk.Size -le 0) {
        throw 'Refusing partition/format: disk is not the newly created bounded VHD'
    }
    @"
select disk $($ownedDisk.Number)
create partition primary
format fs=ntfs quick label="$label"
assign letter=$drive
"@ | Set-Content -Encoding ascii $diskScript
    diskpart.exe /s $diskScript
    if ($LASTEXITCODE -ne 0) { throw 'Could not format the owned bounded VHD' }
    $volume = Get-Volume -DriveLetter $drive
    if ($volume.FileSystemLabel -ne $label -or $volume.Size -ge 64MB -or $volume.Size -le 0) {
        throw 'Refusing to write: the drive is not the newly created bounded fixture'
    }
    $env:WOS_TEST_FULL_DISK_DIRECTORY = "${drive}:\"
    $testArguments = @('-test.v', '-test.run=^TestReturnResolutionCleanupRecoversRealFullDisk$', '-test.timeout=60s')
    $output = & $testBinary @testArguments 2>&1
    $testExit = $LASTEXITCODE
    $output | ForEach-Object { Write-Output $_ }
    if ($testExit -ne 0 -or -not ($output -match '^--- PASS: TestReturnResolutionCleanupRecoversRealFullDisk')) {
        throw 'Native Windows full-disk resolution recovery did not pass'
    }
    New-Item -ItemType Directory -Force dist | Out-Null
    $proof = [ordered]@{
        schema = 1
        commit = (git rev-parse HEAD).Trim()
        native_host = 'windows'
        isolated_vhd_mib = 32
        passed = $true
        phases = @('proof_persistence', 'draft_archived', 'archive_persisted', 'draft_unfrozen')
    }
    $proof | ConvertTo-Json -Depth 4 | Set-Content -Encoding utf8 dist/full-disk-windows-amd64.json
} finally {
    $env:WOS_TEST_FULL_DISK_DIRECTORY = $previousFixture
    if (Test-Path $vhd) {
        @"
select vdisk file="$vhd"
detach vdisk
"@ | Set-Content -Encoding ascii $diskScript
        diskpart.exe /s $diskScript
        if ($LASTEXITCODE -ne 0) { throw "Could not detach owned fixture $vhd" }
    }
    Remove-Item -LiteralPath $fixtureRoot -Recurse -Force
}
