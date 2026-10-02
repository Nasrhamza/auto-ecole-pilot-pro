$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$output = Join-Path $PSScriptRoot '..\assets\logo.png'
$bitmap = [System.Drawing.Bitmap]::new(1024, 1024)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.Clear([System.Drawing.Color]::FromArgb(16, 29, 44))

$orange = [System.Drawing.SolidBrush]::new([System.Drawing.Color]::FromArgb(255, 107, 53))
$white = [System.Drawing.SolidBrush]::new([System.Drawing.Color]::White)
$navy = [System.Drawing.SolidBrush]::new([System.Drawing.Color]::FromArgb(16, 29, 44))
$whitePen = [System.Drawing.Pen]::new([System.Drawing.Color]::White, 40)
$orangePen = [System.Drawing.Pen]::new([System.Drawing.Color]::FromArgb(255, 107, 53), 48)
$orangePen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
$orangePen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round

try {
    $graphics.FillEllipse($orange, 162, 162, 700, 700)
    $graphics.FillEllipse($navy, 238, 238, 548, 548)
    $graphics.DrawEllipse($whitePen, 310, 310, 404, 404)
    $graphics.FillEllipse($white, 445, 445, 134, 134)
    $graphics.DrawLine($whitePen, 512, 512, 376, 632)
    $graphics.DrawLine($whitePen, 512, 512, 648, 632)
    $graphics.DrawLine($whitePen, 512, 512, 512, 330)
    $graphics.DrawLine($orangePen, 320, 792, 704, 792)

    $font = [System.Drawing.Font]::new('Segoe UI', 76, [System.Drawing.FontStyle]::Bold)
    $format = [System.Drawing.StringFormat]::new()
    $format.Alignment = [System.Drawing.StringAlignment]::Center
    $graphics.DrawString('PILOT', $font, $white, [System.Drawing.RectangleF]::new(180, 842, 664, 100), $format)
    $font.Dispose()
    $format.Dispose()

    $bitmap.Save($output, [System.Drawing.Imaging.ImageFormat]::Png)
}
finally {
    $orangePen.Dispose(); $whitePen.Dispose()
    $orange.Dispose(); $white.Dispose(); $navy.Dispose()
    $graphics.Dispose(); $bitmap.Dispose()
}

Write-Output "Auto-ecole logo created: $output"
