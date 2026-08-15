# ─────────────────────────────────────────────────────────────────
# NEXORA – Generate Test Media Library
# ─────────────────────────────────────────────────────────────────
# Creates a realistic folder hierarchy with dummy video, image,
# and subtitle files.  Names are Arabic, English, mixed, and
# include edge-cases like extended characters, Arabic numerals,
# right-to-left marks, and unicode combining characters.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File generate_test_media.ps1
#
# The root of the test library is created at:
#   <script-dir>\test_media_root
# ─────────────────────────────────────────────────────────────────

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$root = Join-Path $PSScriptRoot "test_media_root"

# ── Helper: create a directory and return its path ──────────────
function New-Dir {
    param([string]$Path)
    if (-not (Test-Path $Path)) { New-Item -ItemType Directory -Path $Path -Force | Out-Null }
    return $Path
}

# ── Helper: create a dummy file with some bytes ─────────────────
function New-DummyFile {
    param(
        [string]$Path,
        [int]$SizeKB = 1
    )
    $dir = Split-Path $Path -Parent
    New-Dir $dir | Out-Null

    # Write a small header so the file is non-empty
    $bytes = New-Object byte[] ($SizeKB * 1024)
    # Put a simple identifier in the first few bytes
    $header = [System.Text.Encoding]::UTF8.GetBytes("NEXORA_TEST_FILE")
    [Array]::Copy($header, $bytes, [Math]::Min($header.Length, $bytes.Length))
    [System.IO.File]::WriteAllBytes($Path, $bytes)
}

# ── Helper: create a dummy subtitle (.srt) ──────────────────────
function New-SubtitleFile {
    param([string]$Path)
    $dir = Split-Path $Path -Parent
    New-Dir $dir | Out-Null
    $content = @"
1
00:00:01,000 --> 00:00:04,000
هذا ملف ترجمة تجريبي
This is a test subtitle file

2
00:00:05,000 --> 00:00:09,000
NEXORA Test Data
بيانات اختبار نيكسورا
"@
    [System.IO.File]::WriteAllText($Path, $content, [System.Text.Encoding]::UTF8)
}

# ── Helper: create a tiny placeholder image (1x1 PNG) ───────────
function New-PlaceholderImage {
    param([string]$Path)
    $dir = Split-Path $Path -Parent
    New-Dir $dir | Out-Null
    # Minimal valid 1×1 red PNG (67 bytes)
    $pngBytes = [Convert]::FromBase64String(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
    )
    [System.IO.File]::WriteAllBytes($Path, $pngBytes)
}

Write-Host "`n╔══════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  NEXORA - مولّد بيانات الاختبار                  ║" -ForegroundColor Cyan
Write-Host "║  Test Media Library Generator                    ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════╝`n" -ForegroundColor Cyan

# Clean previous run
if (Test-Path $root) {
    Write-Host "[*] Cleaning previous test data..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $root
}

New-Dir $root | Out-Null
Write-Host "[+] Root: $root`n" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 1. MOVIES  –  أفلام
# ═════════════════════════════════════════════════════════════════
$movies = New-Dir "$root\Movies"

# ── English movie (standard naming) ─────────────────────────────
$m1 = New-Dir "$movies\Inception (2010)"
New-DummyFile "$m1\Inception.2010.1080p.BluRay.x264.mp4" 4
New-PlaceholderImage "$m1\poster.jpg"
New-PlaceholderImage "$m1\banner.jpg"
New-SubtitleFile "$m1\Inception.2010.1080p.BluRay.x264.srt"
New-SubtitleFile "$m1\Inception.2010.Arabic.srt"

# ── Arabic movie name ───────────────────────────────────────────
$m2 = New-Dir "$movies\ذيب (2014)"
New-DummyFile "$m2\ذيب.2014.1080p.mkv" 3
New-PlaceholderImage "$m2\غلاف.jpg"
New-SubtitleFile "$m2\ذيب.2014.عربي.srt"

# ── Mixed Arabic-English movie name ─────────────────────────────
$m3 = New-Dir "$movies\The Message - الرسالة (1976)"
New-DummyFile "$m3\The.Message.1976.720p.mp4" 2
New-PlaceholderImage "$m3\poster.jpg"
New-SubtitleFile "$m3\The.Message.1976.English.srt"
New-SubtitleFile "$m3\الرسالة.١٩٧٦.عربي.srt"

# ── Movie with special characters in name ───────────────────────
$m4 = New-Dir "$movies\Spider-Man - No Way Home (2021)"
New-DummyFile "$m4\Spider-Man.No.Way.Home.2021.4K.UHD.HEVC.mkv" 8
New-PlaceholderImage "$m4\poster.jpg"

# ── Movie with extended/accented characters ─────────────────────
$m5 = New-Dir "$movies\Amélie (2001)"
New-DummyFile "$m5\Amélie.2001.1080p.BluRay.mp4" 3

# ── Movie with Arabic digits (٠١٢٣) ────────────────────────────
$m6 = New-Dir "$movies\فيلم تجريبي ٢٠٢٣"
New-DummyFile "$m6\فيلم.تجريبي.٢٠٢٣.١٠٨٠p.mp4" 2
New-PlaceholderImage "$m6\بوستر.png"

# ── Movie with very long name ───────────────────────────────────
$m7 = New-Dir "$movies\The Shawshank Redemption - الخلاص من شاوشانك (1994)"
New-DummyFile "$m7\The.Shawshank.Redemption.1994.1080p.BluRay.x265.HEVC.mkv" 5
New-SubtitleFile "$m7\الخلاص.من.شاوشانك.عربي.srt"

# ── Movie with brackets and unusual separators ──────────────────
$m8 = New-Dir "$movies\[BDRip] Interstellar (2014) [4K]"
New-DummyFile "$m8\Interstellar.2014.4K.BDRip.HEVC.DTS.mkv" 6
New-PlaceholderImage "$m8\cover.png"

Write-Host "[+] Movies: 8 titles created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 2. SERIES  –  مسلسلات
# ═════════════════════════════════════════════════════════════════
$series = New-Dir "$root\مسلسلات"

# ── English series with seasons ─────────────────────────────────
$s1 = New-Dir "$series\Breaking Bad"
for ($season = 1; $season -le 3; $season++) {
    $sDir = New-Dir "$s1\Season $($season.ToString('D2'))"
    for ($ep = 1; $ep -le 5; $ep++) {
        $epStr = $ep.ToString('D2')
        New-DummyFile "$sDir\Breaking.Bad.S$($season.ToString('D2'))E$epStr.1080p.mkv" 2
    }
}
New-PlaceholderImage "$s1\poster.jpg"
New-PlaceholderImage "$s1\banner.jpg"

# ── Arabic series ───────────────────────────────────────────────
$s2 = New-Dir "$series\العاصوف"
for ($season = 1; $season -le 2; $season++) {
    $sDir = New-Dir "$s2\الموسم $season"
    for ($ep = 1; $ep -le 4; $ep++) {
        New-DummyFile "$sDir\العاصوف موسم $season حلقة $ep 1080p.mp4" 2
    }
}
New-PlaceholderImage "$s2\غلاف.jpg"

# ── Mixed series name with gap (missing episode for testing) ────
$s3 = New-Dir "$series\Game of Thrones - صراع العروش"
$s3s1 = New-Dir "$s3\Season 01"
New-DummyFile "$s3s1\Game.of.Thrones.S01E01.720p.mp4" 2
New-DummyFile "$s3s1\Game.of.Thrones.S01E02.720p.mp4" 2
# Skip E03 intentionally — to test missing episode detection!
New-DummyFile "$s3s1\Game.of.Thrones.S01E04.720p.mp4" 2
New-DummyFile "$s3s1\Game.of.Thrones.S01E05.720p.mp4" 2
New-SubtitleFile "$s3s1\Game.of.Thrones.S01E01.Arabic.srt"

# ── Series with Arabic digits in folder names ───────────────────
$s4 = New-Dir "$series\مسلسل الاختيار"
$s4s1 = New-Dir "$s4\الموسم ١"
for ($ep = 1; $ep -le 3; $ep++) {
    $arabicEp = switch($ep) { 1 {"١"} 2 {"٢"} 3 {"٣"} }
    New-DummyFile "$s4s1\الاختيار الحلقة $arabicEp ١٠٨٠p.mp4" 2
}

# ── Korean drama (extended characters) ──────────────────────────
$s5 = New-Dir "$series\Squid Game - لعبة الحبّار"
$s5s1 = New-Dir "$s5\Season 01"
for ($ep = 1; $ep -le 4; $ep++) {
    New-DummyFile "$s5s1\Squid.Game.S01E$($ep.ToString('D2')).1080p.WEB-DL.mkv" 2
}

Write-Host "[+] Series: 5 titles created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 3. ANIME  –  أنمي
# ═════════════════════════════════════════════════════════════════
$anime = New-Dir "$root\Anime - أنمي"

# ── One Piece (long-running, Arabic filename style) ─────────────
$a1 = New-Dir "$anime\ون بيس - One Piece"
$a1s1 = New-Dir "$a1\الحلقات ١-٥٠"
for ($ep = 1; $ep -le 10; $ep++) {
    New-DummyFile "$a1s1\ون بيس الحلقة $ep 720p.mp4" 1
}
# Add some with Arabic digits
New-DummyFile "$a1s1\ون بيس الحلقة ١١ 720p.mp4" 1
New-DummyFile "$a1s1\ون بيس الحلقة ١٢ 720p.mp4" 1
New-PlaceholderImage "$a1\poster.jpg"

# ── Attack on Titan (standard English naming) ───────────────────
$a2 = New-Dir "$anime\Attack on Titan - هجوم العمالقة"
for ($season = 1; $season -le 2; $season++) {
    $sDir = New-Dir "$a2\Season $($season.ToString('D2'))"
    for ($ep = 1; $ep -le 5; $ep++) {
        New-DummyFile "$sDir\Attack.on.Titan.S$($season.ToString('D2'))E$($ep.ToString('D2')).1080p.mkv" 2
    }
}
New-PlaceholderImage "$a2\poster.jpg"
New-PlaceholderImage "$a2\banner.jpg"

# ── Naruto with mixed naming conventions ────────────────────────
$a3 = New-Dir "$anime\ناروتو شيبودن"
$a3s1 = New-Dir "$a3\الموسم الأول"
for ($ep = 1; $ep -le 6; $ep++) {
    New-DummyFile "$a3s1\Naruto.Shippuden.Episode.$ep.720p.mp4" 1
}
New-SubtitleFile "$a3s1\Naruto.Shippuden.Episode.1.Arabic.srt"

# ── Death Note with episode-only naming ─────────────────────────
$a4 = New-Dir "$anime\Death Note - مذكرة الموت"
for ($ep = 1; $ep -le 8; $ep++) {
    New-DummyFile "$a4\Death.Note.EP$($ep.ToString('D2')).1080p.BDRip.mkv" 2
}
New-PlaceholderImage "$a4\poster.jpg"

# ── Demon Slayer with Unicode combining characters ──────────────
$a5 = New-Dir "$anime\Demon Slayer - قاتل الشياطين"
$a5s1 = New-Dir "$a5\Season 01 - الموسم الأوّل"
for ($ep = 1; $ep -le 4; $ep++) {
    New-DummyFile "$a5s1\Kimetsu.no.Yaiba.S01E$($ep.ToString('D2')).1080p.mp4" 2
}

# ── Dragon Ball Z with Arabic + noise tokens ────────────────────
$a6 = New-Dir "$anime\Dragon Ball Z - دراغون بول زد"
$a6s1 = New-Dir "$a6\الموسم ١"
for ($ep = 1; $ep -le 5; $ep++) {
    New-DummyFile "$a6s1\Dragon.Ball.Z.S01E$($ep.ToString('D2')).720p.x264.AAC.DL.mp4" 1
}

Write-Host "[+] Anime: 6 titles created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 4. DOCUMENTARIES  –  وثائقيات
# ═════════════════════════════════════════════════════════════════
$docs = New-Dir "$root\وثائقيات - Documentaries"

$d1 = New-Dir "$docs\Planet Earth II (2016)"
for ($ep = 1; $ep -le 3; $ep++) {
    New-DummyFile "$d1\Planet.Earth.II.S01E$($ep.ToString('D2')).4K.mkv" 4
}

$d2 = New-Dir "$docs\الجزيرة الوثائقية - وثائقي الحضارة الإسلامية"
for ($ep = 1; $ep -le 3; $ep++) {
    New-DummyFile "$d2\الحضارة الإسلامية الحلقة $ep 1080p.mp4" 2
}

$d3 = New-Dir "$docs\Cosmos - الكون"
New-DummyFile "$d3\Cosmos.A.Spacetime.Odyssey.S01E01.1080p.mkv" 3
New-DummyFile "$d3\Cosmos.A.Spacetime.Odyssey.S01E02.1080p.mkv" 3

Write-Host "[+] Documentaries: 3 titles created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 5. CARTOONS  –  كرتون
# ═════════════════════════════════════════════════════════════════
$cartoons = New-Dir "$root\كرتون"

$c1 = New-Dir "$cartoons\توم وجيري - Tom and Jerry"
for ($ep = 1; $ep -le 5; $ep++) {
    New-DummyFile "$c1\Tom.and.Jerry.EP$($ep.ToString('D2')).720p.mp4" 1
}

$c2 = New-Dir "$cartoons\سبونج بوب"
for ($ep = 1; $ep -le 4; $ep++) {
    New-DummyFile "$c2\SpongeBob.S01E$($ep.ToString('D2')).480p.avi" 1
}

$c3 = New-Dir "$cartoons\كونان - Detective Conan"
for ($ep = 1; $ep -le 6; $ep++) {
    New-DummyFile "$c3\Detective.Conan.EP$($ep.ToString('D3')).720p.mp4" 1
}
# Intentionally add a duplicate with different naming
New-DummyFile "$c3\المحقق كونان الحلقة ١ 720p.mp4" 1

Write-Host "[+] Cartoons: 3 titles created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 6. EDGE CASES  –  حالات خاصة
# ═════════════════════════════════════════════════════════════════
$edge = New-Dir "$root\_Edge Cases - حالات خاصة"

# ── Very long file and folder name ──────────────────────────────
$longName = "This.Is.An.Extremely.Long.Movie.Name.That.Tests.The.System.Limits.2024.4K.UHD.HDR.DV.HEVC.DTS-HD.MA.7.1.Atmos.Remux-GroupName"
New-DummyFile "$edge\$longName.mkv" 2

# ── File with lots of dots and hyphens ──────────────────────────
New-DummyFile "$edge\[Sub-Group]_Anime_Name_-_01v2_[BD_1080p][FLAC][A1B2C3D4].mkv" 2

# ── File with only Arabic name, no English at all ───────────────
New-DummyFile "$edge\فيلم عربي بدون اسم انجليزي ١٠٨٠بي.mp4" 1

# ── File with right-to-left and left-to-right mixing ────────────
New-DummyFile "$edge\Movie مسلسل 2024 الموسم S02E05 حلقة.mkv" 1

# ── File with special unicode chars (accents, diacritics) ───────
New-DummyFile "$edge\Crème Brûlée - The Movie (2023).mp4" 1
New-DummyFile "$edge\München Nights [2022] 4K HDR.mkv" 1

# ── Zero-byte file (corrupted file test) ────────────────────────
$corruptedPath = "$edge\corrupted_file_test.mp4"
New-Dir (Split-Path $corruptedPath -Parent) | Out-Null
[System.IO.File]::WriteAllBytes($corruptedPath, @())

# ── File with Arabic extension description ──────────────────────
New-DummyFile "$edge\عنوان_الفيلم.بجودة.عالية.2024.mkv" 1

# ── Multiple quality versions of same movie (duplicate test) ────
$dupDir = New-Dir "$edge\Duplicates Test - اختبار التكرار"
New-DummyFile "$dupDir\Avengers.Endgame.2019.720p.mp4" 2
New-DummyFile "$dupDir\Avengers.Endgame.2019.1080p.mp4" 4
New-DummyFile "$dupDir\Avengers.Endgame.2019.4K.mkv" 8

# ── Files in nested deeply ──────────────────────────────────────
$deepDir = New-Dir "$edge\مجلد عميق\المستوى الثاني\Third Level\المستوى الرابع\Level Five"
New-DummyFile "$deepDir\deep_nested_video.1080p.mp4" 1

# ── Folder with mixed RTL+LTR in name ──────────────────────────
$rtlDir = New-Dir "$edge\2024 أفلام Action أكشن Movies"
New-DummyFile "$rtlDir\Test.Movie.2024.1080p.mkv" 1

# ── File with Persian/Farsi digits (۰۱۲۳) ──────────────────────
New-DummyFile "$edge\فیلم ۲۰۲۴ با کیفیت ۱۰۸۰.mp4" 1

# ── Non-video files mixed in (should be ignored by scanner) ─────
$junkDir = New-Dir "$edge\ملفات مختلطة - Mixed Files"
New-DummyFile "$junkDir\readme.txt" 1
New-DummyFile "$junkDir\notes.pdf" 1
New-DummyFile "$junkDir\thumbs.db" 1
New-DummyFile "$junkDir\real_video.mp4" 1
New-PlaceholderImage "$junkDir\sample.jpg"
New-SubtitleFile "$junkDir\subtitle.srt"

# ── WMV and rare extensions ─────────────────────────────────────
New-DummyFile "$edge\old_movie.wmv" 1
New-DummyFile "$edge\recording.webm" 1
New-DummyFile "$edge\broadcast.ts" 1
New-DummyFile "$edge\bluray_rip.m2ts" 2
New-DummyFile "$edge\classic.avi" 1
New-DummyFile "$edge\apple_format.m4v" 1
New-DummyFile "$edge\quicktime.mov" 1

Write-Host "[+] Edge cases: 20+ files created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# 7. COVER ART / POSTERS  –  أغلفة
# ═════════════════════════════════════════════════════════════════
$covers = New-Dir "$root\أغلفة - Covers"
New-PlaceholderImage "$covers\generic_poster_01.jpg"
New-PlaceholderImage "$covers\generic_poster_02.png"
New-PlaceholderImage "$covers\بوستر_عربي_01.jpg"
New-PlaceholderImage "$covers\بوستر_عربي_02.png"
New-PlaceholderImage "$covers\banner_wide_01.jpg"
New-PlaceholderImage "$covers\ﺑﺎﻧﺮ_ﻋﺮﻳﺾ.jpg"

Write-Host "[+] Cover art: 6 images created" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════
# Summary
# ═════════════════════════════════════════════════════════════════

# Count totals
$totalFiles = (Get-ChildItem -Path $root -Recurse -File).Count
$totalDirs  = (Get-ChildItem -Path $root -Recurse -Directory).Count
$totalSize  = (Get-ChildItem -Path $root -Recurse -File | Measure-Object -Property Length -Sum).Sum

Write-Host ""
Write-Host "═══════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  ✅ Test media library generated successfully!" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  📁 Root:         $root"
Write-Host "  📂 Directories:  $totalDirs"
Write-Host "  📄 Files:        $totalFiles"
Write-Host "  💾 Total Size:   $([Math]::Round($totalSize / 1KB, 1)) KB"
Write-Host ""
Write-Host "  To test with the NEXORA server, either:" -ForegroundColor Yellow
Write-Host "    1. Set NEXORA_MEDIA_ROOTS=$root" -ForegroundColor Yellow
Write-Host "    2. Use the /api/scan?root=$root endpoint" -ForegroundColor Yellow
Write-Host "    3. POST to /api/index with {`"roots`": [`"$($root -replace '\\','\\')`"]}" -ForegroundColor Yellow
Write-Host ""
