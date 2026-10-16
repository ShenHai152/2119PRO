param(
    [ValidateRange(0, 255)]
    [int]$Amplitude = 255,
    [Alias("Left")]
    [ValidateRange(-1, 255)]
    [int]$Stick = -1,
    [Alias("Right")]
    [ValidateRange(-1, 255)]
    [int]$Throttle = -1,
    [ValidateRange(1, 30)]
    [int]$Seconds = 3,
    [string]$HostAddress = "127.0.0.1",
    [ValidateRange(1, 65535)]
    [int]$Port = 21190
)

$client = [System.Net.Sockets.UdpClient]::new()
$client.Connect($HostAddress, $Port)
$sequence = [uint32]1

if ($Stick -lt 0) { $Stick = $Amplitude }
if ($Throttle -lt 0) { $Throttle = $Amplitude }

function Send-PXNFPacket([byte]$Command, [byte]$StickLevel, [byte]$ThrottleLevel, [uint16]$Duration, [uint32]$Sequence) {
    $packet = [byte[]]::new(16)
    [System.Text.Encoding]::ASCII.GetBytes("PXNF").CopyTo($packet, 0)
    $packet[4] = 1
    $packet[5] = $Command
    $packet[6] = $StickLevel
    $packet[7] = $ThrottleLevel
    [System.BitConverter]::GetBytes($Duration).CopyTo($packet, 8)
    [System.BitConverter]::GetBytes($Sequence).CopyTo($packet, 12)
    [void]$client.Send($packet, $packet.Length)
}

try {
    $end = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $end) {
        Send-PXNFPacket 1 ([byte]$Stick) ([byte]$Throttle) 250 $sequence
        $sequence++
        Start-Sleep -Milliseconds 50
    }
}
finally {
    Send-PXNFPacket 2 0 0 0 $sequence
    $client.Dispose()
}
