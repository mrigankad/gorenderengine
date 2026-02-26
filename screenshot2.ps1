Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class WinHelper2 {
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int n);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, ref RECT r);
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int Left, Top, Right, Bottom; }
}
"@
$procs = Get-Process | Where-Object { $_.MainWindowTitle -like '*Triangle*' -or $_.MainWindowTitle -like '*Render*' }
$proc = $procs | Select-Object -First 1
if ($proc) {
    $hwnd = [IntPtr]$proc.MainWindowHandle
    [WinHelper2]::ShowWindow($hwnd, 9)
    [WinHelper2]::SetForegroundWindow($hwnd)
    Start-Sleep -Milliseconds 800
    $r = New-Object WinHelper2+RECT
    [WinHelper2]::GetWindowRect($hwnd, [ref]$r)
    $w = $r.Right - $r.Left
    $h = $r.Bottom - $r.Top
    Write-Host "Window: ${w}x${h} at ($($r.Left),$($r.Top))"
    if ($w -gt 0 -and $h -gt 0) {
        $bmp = New-Object System.Drawing.Bitmap($w, $h)
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        $g.CopyFromScreen($r.Left, $r.Top, 0, 0, [System.Drawing.Size]::new($w, $h))
        $g.Dispose()
        $path = "C:\Users\E36250409\Desktop\Render Engine -Go\screenshot_window.png"
        $bmp.Save($path)
        $bmp.Dispose()
        Write-Host "Saved to $path"
    }
} else {
    Write-Host "Window not found"
    Get-Process | Where-Object { $_.MainWindowTitle -ne '' } | Select-Object Name, MainWindowTitle
}
