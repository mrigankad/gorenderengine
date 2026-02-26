Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class WinHelper {
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int n);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, ref RECT r);
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int Left, Top, Right, Bottom; }
}
"@
$proc = Get-Process | Where-Object { $_.MainWindowTitle -like '*Triangle*' }
if ($proc) {
    $hwnd = $proc.MainWindowHandle
    [WinHelper]::ShowWindow($hwnd, 9)
    [WinHelper]::SetForegroundWindow($hwnd)
    Start-Sleep -Milliseconds 800
    $r = New-Object WinHelper+RECT
    [WinHelper]::GetWindowRect($hwnd, [ref]$r)
    $w = $r.Right - $r.Left
    $h = $r.Bottom - $r.Top
    $bmp = New-Object System.Drawing.Bitmap($w, $h)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($r.Left, $r.Top, 0, 0, [System.Drawing.Size]::new($w, $h))
    $g.Dispose()
    $path = "C:\Users\E36250409\Desktop\Render Engine -Go\screenshot_window.png"
    $bmp.Save($path)
    $bmp.Dispose()
    Write-Host "Saved to $path (${w}x${h})"
} else {
    Write-Host "Window not found"
}
