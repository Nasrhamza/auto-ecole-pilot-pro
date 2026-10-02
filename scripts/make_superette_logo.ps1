param([string]$OutputPng = "assets\logo.png")

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$size = 512
$bitmap = [System.Drawing.Bitmap]::new($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit

try {
    $graphics.Clear([System.Drawing.Color]::Transparent)
    $rect = [System.Drawing.RectangleF]::new(12, 12, 488, 488)
    $path = [System.Drawing.Drawing2D.GraphicsPath]::new()
    $radius = 92
    $path.AddArc($rect.X, $rect.Y, $radius, $radius, 180, 90)
    $path.AddArc($rect.Right - $radius, $rect.Y, $radius, $radius, 270, 90)
    $path.AddArc($rect.Right - $radius, $rect.Bottom - $radius, $radius, $radius, 0, 90)
    $path.AddArc($rect.X, $rect.Bottom - $radius, $radius, $radius, 90, 90)
    $path.CloseFigure()
    $gradient = [System.Drawing.Drawing2D.LinearGradientBrush]::new($rect, [System.Drawing.Color]::FromArgb(8,28,36), [System.Drawing.Color]::FromArgb(0,168,132), 45)
    $graphics.FillPath($gradient, $path)

    $white = [System.Drawing.Pen]::new([System.Drawing.Color]::White, 24)
    $white.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
    $white.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
    $green = [System.Drawing.SolidBrush]::new([System.Drawing.Color]::FromArgb(50,230,170))
    $graphics.DrawLine($white, 112, 132, 151, 132)
    $graphics.DrawLine($white, 151, 132, 184, 305)
    $graphics.DrawLine($white, 184, 305, 385, 305)
    $graphics.DrawLine($white, 174, 191, 403, 191)
    $graphics.DrawLine($white, 403, 191, 374, 272)
    $graphics.FillEllipse($green, 190, 337, 55, 55)
    $graphics.FillEllipse($green, 335, 337, 55, 55)
    $font = [System.Drawing.Font]::new('Segoe UI', 54, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
    $format = [System.Drawing.StringFormat]::new()
    $format.Alignment = [System.Drawing.StringAlignment]::Center
    $graphics.DrawString('PRO', $font, [System.Drawing.Brushes]::White, [System.Drawing.RectangleF]::new(96, 407, 320, 72), $format)

    $full = [System.IO.Path]::GetFullPath($OutputPng)
    [System.IO.Directory]::CreateDirectory([System.IO.Path]::GetDirectoryName($full)) | Out-Null
    $bitmap.Save($full, [System.Drawing.Imaging.ImageFormat]::Png)
}
finally {
    $graphics.Dispose()
    $bitmap.Dispose()
}
