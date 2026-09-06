param(
  [string]$Root = (Join-Path (Split-Path $PSScriptRoot -Parent) "test-media-library")
)

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  NEXORA Showcase Demo Library Generator (استراحة VIP)     " -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor Cyan

$files = @(
  # =========================================================================
  # 1. أفلام سينمائية عالمية بدقة 4K و 1080p وترجمات عربية
  # =========================================================================
  "أفلام\أفلام أجنبية 4K\Oppenheimer (2023)\Oppenheimer.2023.2160p.UHD.HDR.mkv",
  "أفلام\أفلام أجنبية 4K\Oppenheimer (2023)\Oppenheimer.2023.ar.srt",
  "أفلام\أفلام أجنبية 4K\Dune Part Two (2024)\Dune.Part.Two.2024.2160p.WEB-DL.HDR.mkv",
  "أفلام\أفلام أجنبية 4K\Dune Part Two (2024)\Dune.Part.Two.2024.ar.srt",
  "أفلام\أفلام أجنبية 4K\Interstellar (2014)\Interstellar.2014.2160p.IMAX.mkv",
  "أفلام\أفلام أجنبية 4K\Interstellar (2014)\Interstellar.2014.ar.srt",
  "أفلام\أفلام أجنبية 4K\Inception (2010)\Inception.2010.1080p.BluRay.mkv",
  "أفلام\أفلام أجنبية 4K\The Batman (2022)\The.Batman.2022.4K.HDR.mkv",
  "أفلام\أفلام أجنبية 4K\The Fall Guy (2024)\The.Fall.Guy.2024.1080p.WEBRip.mp4",
  "أفلام\أفلام أكشن وحركة\Top Gun Maverick (2022)\Top.Gun.Maverick.2022.2160p.HDR.mkv",
  "أفلام\أفلام أكشن وحركة\Mad Max Fury Road (2015)\Mad.Max.Fury.Road.2015.1080p.BluRay.mkv",
  "أفلام\أفلام أكشن وحركة\Mission Impossible Dead Reckoning (2023)\Mission.Impossible.Dead.Reckoning.2023.4K.mkv",
  "أفلام\أفلام رعب وإثارة\A Quiet Place Day One (2024)\A.Quiet.Place.Day.One.2024.1080p.mkv",
  "أفلام\أفلام رعب وإثارة\The Conjuring (2013)\The.Conjuring.2013.1080p.BluRay.mkv",
  "أفلام\أفلام عالمية وكورية\Parasite (2019)\Parasite.2019.1080p.Korean.mkv",
  "أفلام\أفلام عالمية وكورية\Jawan (2023)\Jawan.2023.1080p.Hindi.mkv",
  "أفلام\أفلام عربية ومصرية\الفيل الأزرق 2 (2019)\الفيل.الأزرق.2.2019.1080p.WEB-DL.mp4",
  "أفلام\أفلام عربية ومصرية\كيرة والجن (2022)\كيرة.والجن.2022.1080p.WEB-DL.mp4",
  "أفلام\أفلام عربية ومصرية\ولاد رزق 3 القاضية (2024)\Welad.Rizk.3.2024.1080p.mp4",
  "أفلام\أفلام عربية ومصرية\سطار (2022)\Sattar.2022.1080p.Saudi.Cinema.mp4",

  # =========================================================================
  # 2. سلاسل سينمائية رسمية (Franchises & Collections)
  # =========================================================================
  "أفلام\سلاسل سينمائية\John Wick Collection\John Wick (2014)\John.Wick.2014.1080p.mkv",
  "أفلام\سلاسل سينمائية\John Wick Collection\John Wick Chapter 2 (2017)\John.Wick.Chapter.2.2017.1080p.mkv",
  "أفلام\سلاسل سينمائية\John Wick Collection\John Wick Chapter 3 Parabellum (2019)\John.Wick.Chapter.3.2019.1080p.mkv",
  "أفلام\سلاسل سينمائية\John Wick Collection\John Wick Chapter 4 (2023)\John.Wick.Chapter.4.2023.2160p.HDR.mkv",
  "أفلام\سلاسل سينمائية\The Dark Knight Trilogy\Batman Begins (2005)\Batman.Begins.2005.1080p.mkv",
  "أفلام\سلاسل سينمائية\The Dark Knight Trilogy\The Dark Knight (2008)\The.Dark.Knight.2008.2160p.mkv",
  "أفلام\سلاسل سينمائية\The Dark Knight Trilogy\The Dark Knight Rises (2012)\The.Dark.Knight.Rises.2012.1080p.mkv",

  # =========================================================================
  # 3. مسلسلات ودراما عالمية ومواسم متعددة (TV Series)
  # =========================================================================
  "مسلسلات\مسلسلات أجنبية\Stranger Things\Season 01\Stranger.Things.S01E01.1080p.NF.mkv",
  "مسلسلات\مسلسلات أجنبية\Stranger Things\Season 01\Stranger.Things.S01E02.1080p.NF.mkv",
  "مسلسلات\مسلسلات أجنبية\Stranger Things\Season 04\Stranger.Things.S04E01.2160p.HDR.mkv",
  "مسلسلات\مسلسلات أجنبية\Stranger Things\Season 04\Stranger.Things.S04E02.2160p.HDR.mkv",
  "مسلسلات\مسلسلات أجنبية\Breaking Bad\Season 01\Breaking.Bad.S01E01.1080p.BluRay.mkv",
  "مسلسلات\مسلسلات أجنبية\Breaking Bad\Season 01\Breaking.Bad.S01E02.1080p.BluRay.mkv",
  "مسلسلات\مسلسلات أجنبية\Breaking Bad\Season 05\Breaking.Bad.S05E01.1080p.BluRay.mkv",
  "مسلسلات\مسلسلات أجنبية\Game of Thrones\Season 01\Game.of.Thrones.S01E01.1080p.mkv",
  "مسلسلات\مسلسلات أجنبية\Game of Thrones\Season 01\Game.of.Thrones.S01E02.1080p.mkv",
  "مسلسلات\مسلسلات أجنبية\The Last of Us\Season 01\The.Last.of.Us.S01E01.2160p.HDR.mkv",
  "مسلسلات\مسلسلات أجنبية\The Last of Us\Season 01\The.Last.of.Us.S01E02.2160p.HDR.mkv",
  "مسلسلات\مسلسلات أجنبية\Silo\Season 01\Silo.S01E01.2160p.WEB-DL.mkv",
  "مسلسلات\مسلسلات أجنبية\Silo\Season 01\Silo.S01E02.2160p.WEB-DL.mkv",
  "مسلسلات\مسلسلات أجنبية\Silo\Season 02\Silo.S02E01.1080p.WEB-DL.mkv",
  "مسلسلات\مسلسلات كورية\Squid Game\Season 01\Squid.Game.S01E01.1080p.Korean.mkv",
  "مسلسلات\مسلسلات كورية\Squid Game\Season 01\Squid.Game.S01E02.1080p.Korean.mkv",
  "مسلسلات\مسلسلات تركية\الطائر الرفراف\الموسم الأول\Yali.Capkin.S01E01.1080p.Turkish.mkv",
  "مسلسلات\مسلسلات تركية\الطائر الرفراف\الموسم الأول\Yali.Capkin.S01E02.1080p.Turkish.mkv",
  "مسلسلات\مسلسلات تركية\قيامة أرطغرل\الموسم الأول\Dirilis.Ertugrul.S01E01.1080p.mkv",

  # =========================================================================
  # 4. رمضانيات ودراما سعودية وخليجية (Ramadan & Saudi Shows)
  # =========================================================================
  "رمضانيات\شباب البومب\الموسم 11\شباب البومب 11 - الحلقة 01.mp4",
  "رمضانيات\شباب البومب\الموسم 11\شباب البومب 11 - الحلقة 02.mp4",
  "رمضانيات\شباب البومب\الموسم 12\شباب البومب 12 - الحلقة 01 1080p.mp4",
  "رمضانيات\شباب البومب\الموسم 12\شباب البومب 12 - الحلقة 02 1080p.mp4",
  "رمضانيات\طاش ما طاش\الموسم 19\طاش 19 - الحلقة 01.mp4",
  "رمضانيات\الحشاشين\الموسم الأول\الحشاشين.S01E01.1080p.mp4",
  "رمضانيات\الحشاشين\الموسم الأول\الحشاشين.S01E02.1080p.mp4",
  "رمضانيات\جعفر العمدة\الموسم 1\جعفر العمدة - الحلقة 01 1080p.mp4",
  "رمضانيات\الاختيار\الموسم 1\الاختيار.S01E01.1080p.mp4",

  # =========================================================================
  # 5. أنمي ورسوم يابانية (Anime)
  # =========================================================================
  "أنمي\Attack on Titan - هجوم العمالقة\Season 01\Attack.on.Titan.S01E01.1080p.mkv",
  "أنمي\Attack on Titan - هجوم العمالقة\Season 01\Attack.on.Titan.S01E02.1080p.mkv",
  "أنمي\Attack on Titan - هجوم العمالقة\Season 04\Attack.on.Titan.S04E01.1080p.mkv",
  "أنمي\One Piece - ون بيس\Season 01\One.Piece.EP.001.720p.mkv",
  "أنمي\One Piece - ون بيس\Season 01\ون بيس الحلقة 002 1080p.mp4",
  "أنمي\Demon Slayer - قاتل الشياطين\Season 01\Demon.Slayer.S01E01.1080p.mkv",
  "أنمي\Demon Slayer - قاتل الشياطين\Season 02\Demon.Slayer.S02E01.1080p.mkv",
  "أنمي\Death Note - مذكرة الموت\Season 01\Death.Note.S01E01.1080p.mkv",
  "أنمي\Spirited Away (2001)\Spirited.Away.2001.1080p.Ghibli.mkv",

  # =========================================================================
  # 6. أطفال وعائلة واستوديوهات ديزني وبيكسار (Kids & Pixar & Disney)
  # =========================================================================
  "أطفال\أفلام بيكسار وديزني\خلطبيطة بصلصة - Ratatouille (2007)\Ratatouille.2007.1080p.Arabic.Dubbed.mkv",
  "أطفال\أفلام بيكسار وديزني\Inside Out 2 (2024)\Inside.Out.2.2024.1080p.Animation.mkv",
  "أطفال\أفلام بيكسار وديزني\Toy Story 4 (2019)\Toy.Story.4.2019.1080p.mkv",
  "أطفال\أفلام بيكسار وديزني\The Lion King (1994)\The.Lion.King.1994.1080p.Arabic.mkv",
  "أطفال\أفلام بيكسار وديزني\Moana (2016)\Moana.2016.1080p.Family.mkv",
  "أطفال\أفلام بيكسار وديزني\Coco (2017)\Coco.2017.1080p.Arabic.mkv",
  "أطفال\كرتون وسبيستون\توم وجيري - Tom and Jerry\Season 01\Tom.and.Jerry.S01E01.720p.mp4",
  "أطفال\كرتون وسبيستون\توم وجيري - Tom and Jerry\Season 01\Tom.and.Jerry.S01E02.720p.mp4",
  "أطفال\كرتون وسبيستون\عهد الأصدقاء\الموسم الأول\عهد الأصدقاء - الحلقة 01.mp4",
  "أطفال\كرتون وسبيستون\القناص - Hunter x Hunter\Season 01\Hunter.x.Hunter.S01E01.720p.mp4",

  # =========================================================================
  # 7. مسرحيات وكوميديا (Plays & Theater)
  # =========================================================================
  "مسرحيات\مدرسة المشاغبين\مدرسة المشاغبين 1973 1080p.mp4",
  "مسرحيات\العيال كبرت\العيال كبرت 1979 1080p.mp4",
  "مسرحيات\سيف العرب\سيف العرب 1992 طارق العلي وحياة الفهد.mp4",
  "مسرحيات\شاهد ماشفش حاجة\شاهد ماشفش حاجة 1976 عادل إمام.mp4",

  # =========================================================================
  # 8. وثائقيات وناشيونال جيوغرافيك (Documentaries & Nature 4K)
  # =========================================================================
  "وثائقيات\Planet Earth III (2023)\Season 01\Planet.Earth.III.S01E01.4K.UHD.mkv",
  "وثائقيات\Planet Earth III (2023)\Season 01\Planet.Earth.III.S01E02.4K.UHD.mkv",
  "وثائقيات\Our Planet (2019)\Season 01\Our.Planet.S01E01.2160p.HDR.mkv",
  "وثائقيات\Free Solo (2018)\Free.Solo.2018.1080p.NatGeo.mkv"
)

$createdCount = 0
foreach ($relative in $files) {
  $path = Join-Path $Root $relative
  $parent = Split-Path $path -Parent
  if (-not (Test-Path -LiteralPath $parent)) { 
    New-Item -ItemType Directory -Path $parent -Force | Out-Null 
  }
  if (-not (Test-Path -LiteralPath $path)) { 
    if ($path.EndsWith(".srt")) {
      Set-Content -LiteralPath $path -Value "1`n00:00:01,000 --> 00:00:05,000`nمرحباً بكم في منصة NEXORA السينمائية الفاخرة للاستراحات." -Encoding UTF8
    } else {
      Set-Content -LiteralPath $path -Value "NEXORA Ultra-HD Demo Media Stream. Format: 4K/1080p. Codec: HEVC/H.264." -NoNewline
    }
    $createdCount++
  }
}

$readme = Join-Path $Root "README-TEST-LIBRARY.md"
$readmeContent = @"
# 🌟 NEXORA VIP Lounge Showcase Library (مكتبة استراحة VIP الفاخرة)

مجلد وسائط متكامل وشامل ومُعد هندسياً لعروض وفهرسة نظام NEXORA للعملاء.
يحتوي على تنظيم احترافي لكافة الأقسام:
- 🎬 أفلام أجنبية وعربية بدقة 4K و 1080p
- 🍿 سلاسل سينمائية رسمية (John Wick, The Dark Knight)
- 📺 مسلسلات عالمية متعددة المواسم (Stranger Things, Breaking Bad, The Last of Us)
- 🌙 رمضانيات ودراما سعودية وخليجية (شباب البومب، طاش ما طاش، الحشاشين)
- 🎌 أنمي ياباني بأعلى تصنيف (هجوم العمالقة، ون بيس، قاتل الشياطين)
- 🧸 أطفال وعائلة وبيكسار وديزني (خلطبيطة بصلصة، Inside Out 2، سبيستون)
- 🎭 مسرحيات وكوميديا كلاسيكية وحديثة (مدرسة المشاغبين، العيال كبرت، طارق العلي)
- 🌍 وثائقيات طبيعة 4K (Planet Earth III, Our Planet)
"@

Set-Content -LiteralPath $readme -Value $readmeContent -Encoding UTF8

Write-Host "✅ تم تجهيز مكتبة العرض الفاخرة بنجاح!" -ForegroundColor Green
Write-Host "📂 المسار: $Root" -ForegroundColor Cyan
Write-Host "📊 إجمالي ملفات وأقسام العرض: $($files.Count) ملف ومنظومة" -ForegroundColor Yellow
