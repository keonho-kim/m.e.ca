// Purpose: Frame typed Windows clipboard reads under native and encoded-memory admission.
// Pattern: Ephemeral Windows PowerShell script with embedded resource guards.
// Usage: The WSL adapter passes this fixed script to the host PowerShell process.
// Related: internal/tui/clipboard/wsl.go, internal/tui/clipboard/wsl_resources.go.
package clipboard

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/media"
)

func windowsImageScript() string {
	return fmt.Sprintf(windowsImageScriptTemplate, media.MaxBytes, media.MaxPixels, workerEmpty, workerLimit, workerFailed, windowsResourceTypes, int64(media.MaxPixels)*8+int64(media.MaxBytes)*2)
}

// Use typed clipboard access only: custom GetData formats in Windows PowerShell's
// .NET Framework can deserialize arbitrary objects. Binary frames avoid Base64.
const windowsImageScriptTemplate = `$ErrorActionPreference = 'Stop'
$MaxBytes = %d
$MaxPixels = %d
$EmptyStatus = %d
$LimitStatus = %d
$FailedStatus = %d
$resourceTypes = @'
%s
'@
$MaxMemory = %d
$out = [Console]::OpenStandardOutput()
function Write-Header([long]$length) {
  $header = New-Object byte[] 9
  for ($i=0; $i -lt 8; $i++) { $header[8-$i] = [byte](($length -shr (8*$i)) -band 255) }
  $out.Write($header,0,9)
}
function Stop-Clipboard([byte]$status) {
  $header = New-Object byte[] 9
  $header[0]=$status
  $out.Write($header,0,9)
  $out.Flush()
  exit 0
}
try {
  Add-Type -TypeDefinition $resourceTypes -Language CSharp
  $guard = New-Object MecaClipboard.MemoryGuard -ArgumentList ([long]$MaxMemory)
  Add-Type -AssemblyName System.Windows.Forms
  Add-Type -AssemblyName System.Drawing
  $paths = [System.Windows.Forms.Clipboard]::GetFileDropList()
  if ($paths.Count -gt 0) {
    $path = $paths[0]
    if ($path.StartsWith('\') -or -not [IO.Path]::IsPathRooted($path)) { Stop-Clipboard $FailedStatus }
    $file = [IO.File]::Open($path,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
    try {
      if ($file.Length -le 0 -or $file.Length -gt $MaxBytes) { Stop-Clipboard $LimitStatus }
      Write-Header $file.Length
      $file.CopyTo($out,65536)
      $out.Flush()
      exit 0
    } finally { $file.Dispose() }
  }
  $img = [System.Windows.Forms.Clipboard]::GetImage()
  if ($null -ne $img) {
    $stream = New-Object MecaClipboard.BoundedStream -ArgumentList ([int]$MaxBytes)
    try {
      if ([long]$img.Width * [long]$img.Height -gt $MaxPixels) { Stop-Clipboard $LimitStatus }
      $img.Save($stream,[System.Drawing.Imaging.ImageFormat]::Png)
      if ($stream.Length -gt $MaxBytes) { Stop-Clipboard $LimitStatus }
      Write-Header $stream.Length
      $stream.Position=0
      $stream.CopyTo($out,65536)
      $out.Flush()
      exit 0
    } catch {
      if ($stream.Limited) { Stop-Clipboard $LimitStatus }
      throw
    } finally { $stream.Dispose(); $img.Dispose() }
  }
  Stop-Clipboard $EmptyStatus
} catch { Stop-Clipboard $FailedStatus }
finally { if ($null -ne $guard) { $guard.Dispose() } }
`
