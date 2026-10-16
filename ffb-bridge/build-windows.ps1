$ErrorActionPreference = 'Stop'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go test -buildvcs=false ./...
go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXN2119FFBBridge.exe .
go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXNMSFSAdapter.exe ./cmd/pxn-msfs-adapter
Write-Host 'Built PXN2119FFBBridge.exe and PXNMSFSAdapter.exe'
