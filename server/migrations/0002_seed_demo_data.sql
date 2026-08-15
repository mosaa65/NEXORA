-- ==============================================================================
-- NEXORA MEDIA LIBRARY - SEED DEMO DATA MIGRATION
-- File: 0002_seed_demo_data.sql
-- Description: Populates initial showcase anime media items, seasons & video files
-- ==============================================================================

-- 1. Seed Media Items (أعمال الأنمي العشرة الرئيسية)
INSERT INTO media_items (id, category_id, title_ar, title_en, type, plot_ar, release_year, rating, poster_path, banner_path, genres) VALUES
    (1, 3, 'طوكيو غول', 'Tokyo Ghoul', 'anime', 'في طوكيو حيث تعيش غيلان بين البشر بالتخفي، تنقلب حياة الشاب (كانيكي) عندما تلتهمه إحدى الغيلان بدلاً من أن تصبح عشاءه، فيتحول إلى نصف بشري ونصف غول محاصر بين عالمين.', 2014, 8.7, '/images/tokyo_ghoul_hero.png', '/images/tokyo_ghoul_hero.png', ARRAY['أنمي', 'فانتازيا مظلمة', 'رعب']),
    (2, 3, 'هجوم العمالقة', 'Attack on Titan', 'anime', 'منذ مائة عام، ظهرت العمالقة فجأة ودمرت معظم البشرية. يعيش الباقون في عالم محاط بأسوار ضخمة لحمايتهم من العمالقة... عندما يُخترق السور الأول، يبدأ إيرين غيغار رحلة الانتقام والبحث عن الحقيقة.', 2013, 9.0, '/images/attack_on_titan_poster.png', '/images/aot_banner_detail.png', ARRAY['خيال مظلم', 'دراما', 'أكشن']),
    (3, 3, 'ديمون سلاير', 'Demon Slayer', 'anime', 'يتعهد تانجيرو كاماتو بالانتقام لعائلته وإعادة أخته نيزوكو إلى هيئتها البشرية بعد تحولها إلى شيطان، منضماً إلى فيلق قتلة الشياطين.', 2023, 9.1, '/images/demon_slayer_poster.png', '/images/demon_slayer_poster.png', ARRAY['أكشن', 'شياطين', 'سيوف']),
    (4, 3, 'جوجوتسو كايسن', 'Jujutsu Kaisen', 'anime', 'ينتلع الفتى إيتادوري يوجي إصبع الساحر الأسطوري ريومن سوكونا، فيصبح وعاءً له وينضم لمدرسة جوجوتسو لمكافحة اللعنات.', 2023, 9.0, '/images/jujutsu_kaisen_poster.png', '/images/jujutsu_kaisen_poster.png', ARRAY['لعنات', 'خوارق', 'معارك']),
    (5, 3, 'ون بيس', 'One Piece', 'anime', 'ينطلق مونكي دي لوفي وطاقم قبعة القش في رحلة أسطورية عبر البحار للبحث عن الكنز الأعظم ون بيس وليصبح ملك القراصنة.', 2023, 9.0, '/images/one_piece_poster.png', '/images/one_piece_poster.png', ARRAY['مغامرة', 'قراصنة', 'كوميديا']),
    (6, 3, 'ون بنش مان', 'One Punch Man', 'anime', 'سايتاما بطل خارق يقضي على أي خصم بلكمة واحدة فقط، ويبحث عن مواجهة حقيقية تعيد له حماس القتال.', 2015, 8.3, '/images/attack_on_titan_poster.png', '/images/attack_on_titan_poster.png', ARRAY['أبطال خارقون', 'كوميديا', 'قتال']),
    (7, 3, 'ناروتو شيبودن', 'Naruto Shippuden', 'anime', 'يعود ناروتو بعد سنوات التدريب يسعى لحماية قريته واستعادة صديقه ساسكي وتحقيق حلمه بلقب الهوكاجي.', 2017, 8.6, '/images/naruto_poster.png', '/images/naruto_poster.png', ARRAY['نينجا', 'صداقة', 'معارك']),
    (8, 3, 'بليتش', 'Bleach', 'anime', 'يحصل إيتشيغو كوروساكي على قوى الشينيغامي لحماية البشر والأرواح من الوحوش الجائعة.', 2012, 8.5, '/images/jujutsu_kaisen_poster.png', '/images/jujutsu_kaisen_poster.png', ARRAY['شينيغامي', 'سيوف', 'أرواح']),
    (9, 3, 'المحقّق كونان', 'Detective Conan', 'anime', 'يتناول سينشي كودو عقاراً يقلص جسده إلى طفل فيتخذ اسم كونان إيدوجاوا ويحل أعقد القضايا أثناء تتبع العصابة السوداء.', 2023, 8.4, '/images/tokyo_ghoul_hero.png', '/images/tokyo_ghoul_hero.png', ARRAY['غموض', 'تحقيق', 'ذكاء']),
    (10, 3, 'دراغون بول سوبر', 'Dragon Ball Super', 'anime', 'يخوض غوكو ومحاربو الزد مواجهات أسطورية ضد آلهة الدمار ومحاربي الأكوان في بطولات القوة.', 2018, 8.4, '/images/demon_slayer_poster.png', '/images/demon_slayer_poster.png', ARRAY['تحولات', 'معارك خارقة', 'أكوان'])
ON CONFLICT (id) DO NOTHING;

-- 2. Seed Seasons (مواسم هجوم العمالقة)
INSERT INTO seasons (id, media_item_id, season_number, title_ar, title_en) VALUES
    (1, 2, 1, 'الموسم الأول', 'Season 01'),
    (2, 2, 2, 'الموسم الثاني', 'Season 02'),
    (3, 2, 3, 'الموسم الثالث', 'Season 03'),
    (4, 2, 4, 'الموسم الرابع', 'Season 04'),
    (5, 2, 5, 'الموسم الأخير', 'Final Season')
ON CONFLICT (id) DO NOTHING;

-- 3. Seed Sample Video Files (عينة ملفات فيديو للمشغل)
INSERT INTO video_files (id, media_item_id, season_id, episode_number, title_ar, title_en, file_path, file_size, duration, resolution) VALUES
    (1, 2, 3, 1, 'الحلقة 01 - مكان إشارة المعركة', 'Episode 01 - Battle Signal', 'C:\Media\Anime\Attack on Titan\S03E01.mkv', 850000000, 1450, '1080p'),
    (2, 2, 3, 2, 'الحلقة 02 - عودة إلى الشكل', 'Episode 02 - Return to Form', 'C:\Media\Anime\Attack on Titan\S03E02.mkv', 860000000, 1450, '1080p'),
    (3, 2, 3, 3, 'الحلقة 03 - ضوء الأمل', 'Episode 03 - Ray of Hope', 'C:\Media\Anime\Attack on Titan\S03E03.mkv', 870000000, 1450, '1080p'),
    (4, 2, 3, 4, 'الحلقة 04 - الليل يتقدم', 'Episode 04 - Night Falls', 'C:\Media\Anime\Attack on Titan\S03E04.mkv', 880000000, 1450, '1080p')
ON CONFLICT (id) DO NOTHING;
