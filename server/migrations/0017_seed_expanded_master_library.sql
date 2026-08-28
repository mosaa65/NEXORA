-- ==============================================================================
-- NEXORA MEDIA LIBRARY - EXPANDED MASTER DEMO DATA & SMART HUBS
-- File: 0017_seed_expanded_master_library.sql
-- Description: Seeds rich Hollywood, Arabic, Disney, Pixar, Anime, Series & Smart Hubs
-- ==============================================================================

-- 1. Insert rich Media Items across Movies, Cartoons, Series, Family, Documentaries & Plays
INSERT INTO media_items (id, category_id, title_ar, title_en, type, plot_ar, release_year, rating, content_rating, poster_path, banner_path, genres) VALUES
    -- === 1. أفلام سينمائية عالمية وعربية (Movies) ===
    (11, 1, 'بين النجوم', 'Interstellar', 'movie', 'في مستقبل بائس حيث تعاني الأرض من مجاعة مدمرة، ينطلق فريق من رواد الفضاء في رحلة عبر ثقب دودي بالقرب من كوكب زحل بحثاً عن موطن جديد للبشرية.', 2014, 8.7, 'PG-13', '/images/interstellar_poster.png', '/images/interstellar_poster.png', ARRAY['خيال علمي', 'دراما', 'مغامرة', 'أجنبي']),
    (12, 1, 'أوبنهايمر', 'Oppenheimer', 'movie', 'سيرة ملحمية تروي قصة الفيزيائي الأمريكي روبرت أوبنهايمر ودوره المحوري في مشروع مانهاتن لتطوير أول قنبلة ذرية في تاريخ البشرية.', 2023, 8.9, 'R', '/images/oppenheimer_poster.png', '/images/oppenheimer_poster.png', ARRAY['تاريخي', 'دراما', 'سيرة ذاتية', 'أجنبي']),
    (13, 1, 'فارس الظلام', 'The Dark Knight', 'movie', 'يتحالف باتمان والضابط جيم جوردون والمدعي العام هارفي دينت لمواجهة الفوضى العارمة التي يشعلها الجوكر في شوارع مدينة غوثام.', 2008, 9.0, 'PG-13', '/images/dark_knight_poster.png', '/images/dark_knight_poster.png', ARRAY['أكشن', 'جريمة', 'دراما', 'أجنبي']),
    (14, 1, 'الفيل الأزرق 2', 'The Blue Elephant 2', 'movie', 'تبدأ أحداث الجزء الثاني بعد 5 سنوات من الجزء الأول، حيث يتم استدعاء الدكتور يحيى راشد لقسم الحالات الخطرة ليجد من يتلاعب بحياته وحياة أسرته.', 2019, 8.1, '18+', '/images/blue_elephant_poster.png', '/images/blue_elephant_poster.png', ARRAY['غموض', 'رعب', 'دراما', 'عربي']),
    (15, 1, 'ولاد رزق 3: القاضية', 'Welad Rizk 3', 'movie', 'بعد سنوات من التفرق والانفصال، يضطر الإخوة للعودة إلى عالم الجريمة والسرقة مرة أخرى في عملية مصيرية ومحفوفة بالمخاطر في قلب الرياض.', 2024, 8.3, '16+', '/images/welad_rizk_poster.png', '/images/welad_rizk_poster.png', ARRAY['أكشن', 'جريمة', 'كوميديا', 'عربي']),
    (16, 1, 'طفيلي', 'Parasite', 'movie', 'تتسلل عائلة كورية فقيرة تدريجياً لتعمل في خدمة عائلة ثرية مرموقة، مما يؤدي إلى سلسلة غير متوقعة من الأحداث المظلمة والمفارقات الاجتماعية.', 2019, 8.6, 'R', '/images/parasite_poster.png', '/images/parasite_poster.png', ARRAY['دراما', 'إثارة', 'كوميديا سوداء', 'كوري']),
    (17, 1, 'توب غان: مافريك', 'Top Gun: Maverick', 'movie', 'بعد أكثر من ثلاثين عاماً من الخدمة كأحد أفضل طياري البحرية، يقود بيت مافريك ميتشل خريجي توب غان في مهمة خاصة خطيرة تتطلب تضحيات جسيمة.', 2022, 8.3, 'PG-13', '/images/topgun_poster.png', '/images/topgun_poster.png', ARRAY['أكشن', 'دراما', 'أجنبي']),
    (18, 1, 'كثيب: الجزء الثاني', 'Dune: Part Two', 'movie', 'يتحد بول أتريدس مع تشاني والفريمن في مسار انتقامي ملحمي ضد المتآمرين الذين دمروا عائلته، بينما يحاول منع مستقبل مرعب يراه في رؤاه.', 2024, 8.6, 'PG-13', '/images/dune2_poster.png', '/images/dune2_poster.png', ARRAY['خيال علمي', 'مغامرة', 'أكشن', 'أجنبي']),

    -- === 2. أفلام كرتون ورسوم متحركة عائلية (Kids & Family Cartoons - Disney, Pixar, DreamWorks) ===
    (19, 4, 'الأسد الملك', 'The Lion King', 'movie', 'النسخة الخالدة لقصة الشبل سيمبا الذي يخوض رحلة النضوج واستعادة حقه في حكم أرض العزة بعد خيانة عمه سكار.', 1994, 8.5, 'G', '/images/lion_king_poster.png', '/images/lion_king_poster.png', ARRAY['كرتون', 'عائلي', 'مغامرة', 'رسوم متحركة', 'ديزني']),
    (20, 4, 'قلباً وقالباً 2', 'Inside Out 2', 'movie', 'تعود مشاعر رايلي في سن المراهقة لمواجهة مشاعر جديدة وغير متوقعة تقتحم المقر الرئيسي، وعلى رأسها القلق والإحراج والغيرة.', 2024, 8.4, 'PG', '/images/inside_out_poster.png', '/images/inside_out_poster.png', ARRAY['كرتون', 'عائلي', 'كوميديا', 'بيكسار', 'ديزني']),
    (21, 4, 'حكاية لعبة 4', 'Toy Story 4', 'movie', 'ينطلق وودي وباز يطير في رحلة طريق شيقة برفقة الأصدقاء ولعبة جديدة تُدعى فوركي، ليكتشفوا مدى اتساع العالم للألعاب الحقيقية.', 2019, 7.7, 'G', '/images/toy_story_poster.png', '/images/toy_story_poster.png', ARRAY['كرتون', 'عائلي', 'مغامرة', 'بيكسار', 'ديزني']),
    (22, 4, 'كوكو', 'Coco', 'movie', 'يطمح الفتى ميغيل لأن يصبح موسيقياً بارعاً كقدوته إرنستو دي لا كروز، ويجد نفسه بشكل غامض في أرض الموتى الملونة لفك لغز تاريخ عائلته العريق.', 2017, 8.4, 'PG', '/images/coco_poster.png', '/images/coco_poster.png', ARRAY['كرتون', 'عائلي', 'موسيقى', 'فانتازيا', 'بيكسار']),
    (23, 4, 'شريك 2', 'Shrek 2', 'movie', 'يسافر شريك والأميرة فيونا والحمار إلى مملكة بعيدة جداً لمقابلة والدي فيونا، لكن الأمور تتعقد بمؤامرات العرابة الطيبة والأمير تشارمينغ.', 2004, 7.3, 'PG', '/images/shrek_poster.png', '/images/shrek_poster.png', ARRAY['كرتون', 'عائلي', 'كوميديا', 'فانتازيا', 'دريم وركس']),
    (24, 4, 'كونغ فو باندا 4', 'Kung Fu Panda 4', 'movie', 'يستعد بو ليصبح المرشد الروحي لوادي السلام، لكنه يواجه الحرباء الشريرة التي تسعى لسرقة فنون القتال من كافة الأبطال السابقين.', 2024, 7.2, 'PG', '/images/kung_fu_panda_poster.png', '/images/kung_fu_panda_poster.png', ARRAY['كرتون', 'عائلي', 'أكشن', 'كوميديا', 'دريم وركس']),
    (25, 4, 'سبايدرمان: بداخل عالم العنكبوت', 'Spider-Man: Into the Spider-Verse', 'movie', 'يصبح المراهق مايلز موراليس سبايدرمان الجديد في نيويورك، ويتعاون مع خمسة أبطال عنكبوتيين من أبعاد موازية لإيقاف تهديد كوني شامل.', 2018, 8.4, 'PG', '/images/spiderman_spiderverse_poster.png', '/images/spiderman_spiderverse_poster.png', ARRAY['كرتون', 'أكشن', 'أبطال خارقون', 'خيال علمي', 'عائلي']),
    (26, 4, 'خلطة بيطة بالصلصة', 'Ratatouille', 'movie', 'فأر موهوب يُدعى ريمي يحلم بأن يصبح طاهياً فرنسياً شهيراً في باريس، ويتحالف سراً مع شاب خجول يعمل في مطعم فاخر.', 2007, 8.1, 'G', '/images/ratatouille_poster.png', '/images/ratatouille_poster.png', ARRAY['كرتون', 'عائلي', 'كوميديا', 'بيكسار', 'ديزني']),

    -- === 3. مسلسلات درامية عربية وتركية وعالمية (Series) ===
    (27, 2, 'قيامة عثمان', 'Kurulus: Osman', 'series', 'سلسلة تاريخية درامية ملحمية تروي سيرة الغازي عثمان بن أرطغرل مؤسس الدولة العثمانية وتحدياته لبناء أمة قوية.', 2023, 8.5, 'TV-14', '/images/osman_poster.png', '/images/osman_poster.png', ARRAY['تاريخي', 'أكشن', 'دراما', 'تركي']),
    (28, 2, 'الحفرة', 'Cukur', 'series', 'تدور الأحداث في حي الحفرة الأكثر خطورة في إسطنبول، حيث تفرض عائلة كوشوفالي سيطرتها، ويعود ياماش كوشوفالي لحماية عائلته وحيه.', 2021, 8.3, 'TV-MA', '/images/cukur_poster.png', '/images/cukur_poster.png', ARRAY['أكشن', 'جريمة', 'دراما', 'تركي']),
    (29, 2, 'جعفر العمدة', 'Gaafar El Omda', 'series', 'رجل أعمال من حي السيدة زينب متزوج من أربع نساء، يعيش مأساة اختطاف ابنه الرضيع منذ 19 عاماً ويسعى طوال حياته لكشف الحقيقة.', 2023, 8.0, 'TV-14', '/images/gaafar_poster.png', '/images/gaafar_poster.png', ARRAY['دراما', 'تشويق', 'عربي']),
    (30, 2, 'أشياء غريبة', 'Stranger Things', 'series', 'عندما يختفي طفل صغير في بلدة هوكينز الغامضة، تكشف البلدة عن سر يتضمن تجارب سرية وقوى خارقة وفتاة صغيرة غير عادية.', 2022, 8.7, 'TV-14', '/images/stranger_things_poster.png', '/images/stranger_things_poster.png', ARRAY['خيال علمي', 'رعب', 'غموض', 'أجنبي']),
    (31, 2, 'لعبة الحبار', 'Squid Game', 'series', 'يواجه مئات اللاعبين الذين يعانون من ضائقة مالية خانقة دعوة غريبة للتنافس في ألعاب أطفال شعبية، لكن الخسارة تعني الموت والجائزة 45 مليار وون.', 2021, 8.0, 'TV-MA', '/images/squid_game_poster.png', '/images/squid_game_poster.png', ARRAY['إثارة', 'غموض', 'دراما', 'كوري']),

    -- === 4. مسلسلات كرتون وأطفال وسبيستون (Kids Cartoons Series) ===
    (32, 4, 'عهد الأصدقاء', 'Romeo no Aoi Sora', 'series', 'القصة المؤثرة للصبي روميو وصديقه ألفريدو ورابطة منظفي المداخن في ميلانو، والبحث عن الأمل والشجاعة في وجه الصعاب.', 1995, 9.1, 'TV-Y7', '/images/romeo_poster.png', '/images/romeo_poster.png', ARRAY['كرتون', 'عائلي', 'دراما', 'سبيستون', 'الزهرة']),
    (33, 4, 'أبطال الديجيتال', 'Digimon Adventure', 'series', 'ينتقل سبعة أطفال إلى العالم الرقمي الغامض ويكتسب كل منهم مرافقاً رقمياً للدفاع عن العالم الرقمي وعالم البشر.', 2000, 8.8, 'TV-Y7', '/images/digimon_poster.png', '/images/digimon_poster.png', ARRAY['كرتون', 'مغامرة', 'خيال علمي', 'سبيستون', 'الزهرة']),
    (34, 4, 'عالم غامبول المدهش', 'The Amazing World of Gumball', 'series', 'مغامرات غامبول القط الأزرق البالغ من العمر 12 عاماً وصديقه العزيز داروين السمكة الذهبية في مدينة إلمور العجيبة.', 2019, 8.4, 'TV-Y7', '/images/gumball_poster.png', '/images/gumball_poster.png', ARRAY['كرتون', 'كوميديا', 'عائلي', 'كرتون نتورك']),

    -- === 5. وثائقيات ومعرفة (Documentaries) ===
    (35, 5, 'كوكب الأرض 3', 'Planet Earth III', 'series', 'السلسلة الوثائقية الأسطورية من بي بي سي التي تستكشف أعظم عجائب الطبيعة وتكيف الكائنات الحية في أصعب بيئات الكوكب.', 2023, 9.3, 'TV-G', '/images/planet_earth_poster.png', '/images/planet_earth_poster.png', ARRAY['طبيعة', 'وثائقي', 'علوم', 'بي بي سي']),
    (36, 5, 'أسرار مقبرة سقارة', 'Secrets of the Saqqara Tomb', 'movie', 'يوثق الفيلم فريقاً من الأثريين المصريين الذين ينقبون في ممرات وأعماق مقبرة كهنوتية لم تُمس لأكثر من 4400 عام.', 2020, 7.8, 'TV-PG', '/images/saqqara_poster.png', '/images/saqqara_poster.png', ARRAY['تاريخ', 'وثائقي', 'حضارات', 'عربي']),

    -- === 6. مسرحيات وكوميديا (Plays) ===
    (37, 6, 'مدرسة المشاغبين', 'Madrast Al-Moshaghbeen', 'movie', 'المسرحية الكوميدية الخالدة عن خمسة طلاب مشاغبين في فصل واحد تتولى معلمة جديدة مهمة تقويم سلوكهم بطرق مبتكرة.', 1973, 8.9, 'TV-PG', '/images/moshaghbeen_poster.png', '/images/moshaghbeen_poster.png', ARRAY['مسرحية', 'كوميديا', 'كلاسيك', 'مصري']),
    (38, 6, 'الزعيم', 'Al Zaeem', 'movie', 'كوميديا سياسية مسرحية ساخرة يجسد فيها عادل إمام دور الشبيه البسيط لرئيس جمهورية ديكتاتور بعد وفاته المفاجئة.', 1993, 8.6, 'TV-PG', '/images/alzaeem_poster.png', '/images/alzaeem_poster.png', ARRAY['مسرحية', 'كوميديا', 'مصري']),

    -- === 7. رمضانيات (Ramadan Specials) ===
    (39, 7, 'الحشاشين', 'The Assassins', 'series', 'ملحمة تاريخية مشوقة تدور في القرن الحادي عشر حول حسن الصباح وفرقة الحشاشين الباطنية وقلعة ألموت الحصينة.', 2024, 8.8, 'TV-14', '/images/assassins_poster.png', '/images/assassins_poster.png', ARRAY['تاريخي', 'دراما', 'إثارة', 'عربي', 'رمضان']),

    -- === 8. مصارعة ورياضة (Wrestling) ===
    (40, 8, 'ريسلمانيا 40', 'WWE WrestleMania XL', 'movie', 'المهرجان الأضخم في تاريخ المصارعة الحرة، المواجهة الأسطورية بين كودي رودز ورومان رينز وذا روك في فيلادلفيا.', 2024, 9.2, 'TV-PG', '/images/wrestlemania_poster.png', '/images/wrestlemania_poster.png', ARRAY['مصارعة', 'رياضة', 'wwe', 'أكشن'])
ON CONFLICT (id) DO UPDATE SET
    title_ar = EXCLUDED.title_ar,
    title_en = EXCLUDED.title_en,
    type = EXCLUDED.type,
    category_id = EXCLUDED.category_id,
    content_rating = EXCLUDED.content_rating,
    genres = EXCLUDED.genres,
    rating = EXCLUDED.rating,
    plot_ar = EXCLUDED.plot_ar;

-- 2. Seed Official Hub Definitions across all categories for SmartHubRail
INSERT INTO hub_definitions (slug, scope, title_ar, title_en, description_ar, accent, icon, rule, priority) VALUES
    -- Hubs in Movies:
    ('movies-disney-pixar', 'movies', '🏰 أفلام ديزني وبيكسار', 'Disney & Pixar Movies', 'روائع الرسوم المتحركة والكرتون السينمائي من ديزني وبيكسار.', 'cyan', 'smile', '{"types":["movie"],"tags_any":["ديزني","بيكسار","كرتون","Disney","Pixar"]}', 110),
    ('movies-top-hollywood', 'movies', '🎬 روائع هوليوود والعالمية', 'Top Hollywood Cinema', 'أقوى إنتاجات السينما العالمية الحائزة على أعلى التقييمات.', 'violet', 'film', '{"types":["movie"],"tags_any":["أجنبي","هوليوود"]}', 95),
    ('movies-arabic-hits', 'movies', '🌟 السينما العربية والمصرية', 'Arabic Blockbusters', 'أضخم أفلام السينما المصرية والخليجية والعربية.', 'rose', 'film', '{"types":["movie"],"tags_any":["عربي","مصري"]}', 90),

    -- Hubs in Kids & Cartoons:
    ('kids-cartoons-world', 'kids', '🎨 روائع الكرتون والأنيميشن', 'World Cartoons Hub', 'أفلام ومسلسلات الكرتون العالمية لجميع الأطفال.', 'amber', 'smile', '{"tags_any":["كرتون","رسوم متحركة","Animation"]}', 120),
    ('kids-spacetoon-golden', 'kids', '⭐ كوكب سبيستون وجيل الطيبين', 'Spacetoon Golden Era', 'ذكريات الطفولة الجميلة والدبلجة السورية الأصيلة لمركز الزهرة.', 'amber', 'spark', '{"tags_any":["سبيستون","الزهرة","كلاسيك"]}', 110),
    ('kids-disney-world', 'kids', '🏰 عوالم ديزني وبيكسار', 'Disney & Pixar Universe', 'سحر ديزني وإبداعات بيكسار بجودة فائقة ودبلجة مميزة.', 'cyan', 'smile', '{"tags_any":["ديزني","بيكسار","Disney","Pixar"]}', 105),
    ('kids-dreamworks', 'kids', '🌙 إبداعات دريم وركس', 'DreamWorks & Illumination', 'مغامرات شريك وكونغ فو باندا والمينيونز الممتعة.', 'emerald', 'smile', '{"tags_any":["دريم وركس","إلومينيشن","DreamWorks"]}', 100),

    -- Hubs in Family Cinema:
    ('family-movie-night', 'family', '🍿 سهرة العائلة وليالي الفشار', 'Family Movie Night', 'أفلام ومسلسلات ممتعة وآمنة لكافة أفراد العائلة.', 'amber', 'smile', '{"tags_any":["عائلي","كرتون","Family"]}', 120),
    ('family-animated-classics', 'family', '🏰 كلاسيكيات الرسوم المتحركة', 'Animated Family Classics', 'التحف الفنية العائلية الخالدة بجودة 4K.', 'cyan', 'smile', '{"types":["movie"],"tags_any":["كرتون","ديزني","بيكسار"]}', 110),

    -- Hubs in Series:
    ('series-turkish-hits', 'series', '👑 روائع الدراما التركية', 'Turkish Blockbusters', 'أضخم الأعمال والمسلسلات التاريخية والدرامية التركية.', 'amber', 'tv', '{"types":["series"],"tags_any":["تركي"]}', 100),
    ('series-arabic-drama', 'series', '🌟 الدراما العربية والخليجية', 'Arabic Top Drama', 'إنتاجات مصرية وخليجية وشامية حصرية ومتميزة.', 'emerald', 'tv', '{"types":["series"],"tags_any":["عربي","مصري"]}', 90),

    -- Hubs in Documentaries:
    ('doc-nature-wildlife', 'documentaries', '🌿 أسرار الطبيعة والحيوان', 'Nature & Wildlife', 'سلاسل بي بي سي وناشيونال جيوغرافيك بجودة تصوير مذهلة.', 'emerald', 'book', '{"tags_any":["طبيعة","حيوان","وثائقي","بي بي سي"]}', 100),
    ('doc-ancient-history', 'documentaries', '🏛️ التاريخ والحضارات المفقودة', 'Ancient Civilizations', 'أسرار الفراعنة والحضارات الإنسانية العريقة.', 'amber', 'book', '{"tags_any":["تاريخ","حضارات","أثريات"]}', 90)
ON CONFLICT (slug) DO UPDATE SET
    title_ar = EXCLUDED.title_ar,
    title_en = EXCLUDED.title_en,
    description_ar = EXCLUDED.description_ar,
    scope = EXCLUDED.scope,
    rule = EXCLUDED.rule,
    accent = EXCLUDED.accent,
    icon = EXCLUDED.icon,
    priority = EXCLUDED.priority;
