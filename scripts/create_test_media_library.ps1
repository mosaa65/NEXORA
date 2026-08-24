param(
  [string]$Root = (Join-Path (Split-Path $PSScriptRoot -Parent) "test-media-library")
)

# Synthetic fixture library for scanner/indexing tests. Files intentionally
# contain small text payloads rather than video bytes: they test parsing,
# grouping, origins, seasons, extensions and safe failed-inspection handling.
$files = @(
  "أفلام\أفلام أجنبية\Dune Part Two (2024)\Dune.Part.Two.2024.2160p.WEB-DL.HDR.Arabic.mkv",
  "أفلام\أفلام أجنبية\Oppenheimer (2023)\Oppenheimer.2023.1080p.BluRay.mkv",
  "أفلام\أفلام أجنبية\The Batman (2022)\The.Batman.2022.4K.HDR.mkv",
  "أفلام\أفلام أجنبية\The Fall Guy (2024)\The.Fall.Guy.2024.1080p.WEBRip.mp4",
  "أفلام\أفلام أكشن\John Wick Chapter 4 (2023)\John.Wick.Chapter.4.2023.2160p.HDR.mkv",
  "أفلام\أفلام أكشن\Mad Max Fury Road (2015)\Mad.Max.Fury.Road.2015.1080p.BluRay.mkv",
  "أفلام\أفلام أكشن\Mission Impossible Dead Reckoning (2023)\Mission.Impossible.Dead.Reckoning.2023.4K.mkv",
  "أفلام\أفلام رعب\A Quiet Place Day One (2024)\A.Quiet.Place.Day.One.2024.1080p.mkv",
  "أفلام\أفلام رعب\The Conjuring (2013)\The.Conjuring.2013.1080p.BluRay.mkv",
  "أفلام\أفلام هندية\Jawan (2023)\Jawan.2023.1080p.Hindi.mkv",
  "أفلام\أفلام كورية\Parasite (2019)\Parasite.2019.1080p.Korean.mkv",
  "أفلام\أفلام عربية\الفيل الأزرق 2 (2019)\الفيل الأزرق 2 2019 1080p.mp4",
  "أطفال\أفلام كرتون\Inside Out 2 (2024)\Inside.Out.2.2024.1080p.Animation.mkv",
  "أطفال\أفلام كرتون\Moana (2016)\Moana.2016.1080p.Family.mkv",
  "أطفال\أفلام كرتون\The Lion King (1994)\The.Lion.King.1994.720p.mkv",
  "أطفال\أفلام كرتون\Toy Story 4 (2019)\Toy.Story.4.2019.1080p.mkv",
  "مسلسلات\مسلسلات أجنبية\Silo\Season 01\Silo.S01E01.2160p.WEB-DL.mkv",
  "مسلسلات\مسلسلات أجنبية\Silo\Season 01\Silo.S01E02.2160p.WEB-DL.mkv",
  "مسلسلات\مسلسلات أجنبية\Silo\Season 02\Silo.S02E01.1080p.WEB-DL.mkv",
  "مسلسلات\مسلسلات أجنبية\The Last of Us\Season 01\The.Last.of.Us.1x01.1080p.HDTV.mkv",
  "مسلسلات\مسلسلات أجنبية\The Last of Us\Season 01\The.Last.of.Us.1x02.1080p.HDTV.mkv",
  "مسلسلات\مسلسلات أجنبية\Breaking Bad\Season 01\Breaking.Bad.S01E01.1080p.BluRay.mkv",
  "مسلسلات\مسلسلات أجنبية\Breaking Bad\Season 01\Breaking.Bad.S01E02.1080p.BluRay.mkv",
  "مسلسلات\مسلسلات تركية\الطائر الرفراف\الموسم الأول\Yali.Capkin.S01E01.1080p.Turkish.mkv",
  "مسلسلات\مسلسلات تركية\الطائر الرفراف\الموسم الأول\Yali.Capkin.S01E02.1080p.Turkish.mkv",
  "مسلسلات\مسلسلات تركية\العشق الممنوع\Season 01\Ask.i.Memnu.S01E01.720p.mkv",
  "مسلسلات\مسلسلات عربية\الاختيار\الموسم 1\الاختيار.S01E01.1080p.mp4",
  "مسلسلات\مسلسلات عربية\الاختيار\الموسم 1\الاختيار.S01E02.1080p.mp4",
  "مسلسلات\مسلسلات كورية\Moving\Season 01\Moving.S01E01.1080p.Korean.mkv",
  "مسلسلات\مسلسلات كورية\Moving\Season 01\Moving.S01E02.1080p.Korean.mkv",
  "أنمي\Attack on Titan - هجوم العمالقة\Season 01\Attack.on.Titan.S01E01.1080p.mkv",
  "أنمي\Attack on Titan - هجوم العمالقة\Season 01\Attack.on.Titan.S01E02.1080p.mkv",
  "أنمي\Attack on Titan - هجوم العمالقة\Season 04\Attack.on.Titan.S04E01.1080p.mkv",
  "أنمي\One Piece - ون بيس\Season 01\One.Piece.EP.001.720p.mkv",
  "أنمي\One Piece - ون بيس\Season 01\ون بيس الحلقة 002 1080p.mp4",
  "أنمي\Demon Slayer\Season 02\Demon.Slayer.S02E01.1080p.mkv",
  "أطفال\مسلسلات كرتون\Tom and Jerry - توم وجيري\Season 01\Tom.and.Jerry.S01E01.720p.mp4",
  "أطفال\مسلسلات كرتون\Tom and Jerry - توم وجيري\Season 01\Tom.and.Jerry.S01E02.720p.mp4",
  "وثائقيات\Planet Earth III\Season 01\Planet.Earth.III.S01E01.4K.mkv",
  "وثائقيات\Planet Earth III\Season 01\Planet.Earth.III.S01E02.4K.mkv",
  "مسرحيات\مدرسة المشاغبين\مدرسة المشاغبين 1973 1080p.mp4",
  "اختبارات متنوعة\Unknown Format\sample.avi",
  "اختبارات متنوعة\Unknown Format\sample.mov",
  "اختبارات متنوعة\Unknown Format\sample.webm",
  "اختبارات متنوعة\Unknown Format\sample.ts",
  "اختبارات متنوعة\Unknown Format\not-a-video.txt"
)

foreach ($relative in $files) {
  $path = Join-Path $Root $relative
  $parent = Split-Path $path -Parent
  if (-not (Test-Path -LiteralPath $parent)) { New-Item -ItemType Directory -Path $parent -Force | Out-Null }
  if (-not (Test-Path -LiteralPath $path)) { Set-Content -LiteralPath $path -Value "NEXORA synthetic media fixture - not a playable video." -NoNewline }
}

$readme = Join-Path $Root "README-TEST-LIBRARY.md"
if (-not (Test-Path -LiteralPath $readme)) {
  Set-Content -LiteralPath $readme -Value "# NEXORA test media library`n`nSynthetic indexing fixtures. Do not use these files for playback or FFmpeg quality tests; they intentionally are not real video streams." -NoNewline
}

Write-Output "Created or retained $($files.Count) synthetic media fixtures under: $Root"
