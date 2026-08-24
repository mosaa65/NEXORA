CREATE TABLE IF NOT EXISTS hub_definitions (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(120) NOT NULL UNIQUE,
    source VARCHAR(20) NOT NULL DEFAULT 'smart' CHECK (source IN ('smart', 'editorial')),
    scope VARCHAR(40) NOT NULL DEFAULT 'all',
    title_ar TEXT NOT NULL,
    title_en TEXT,
    description_ar TEXT,
    description_en TEXT,
    artwork_path TEXT,
    artwork_position VARCHAR(80) NOT NULL DEFAULT 'center center',
    accent VARCHAR(40) NOT NULL DEFAULT 'violet',
    icon VARCHAR(40) NOT NULL DEFAULT 'spark',
    rule JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    min_item_count INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_hub_definitions_scope_active_priority
    ON hub_definitions (scope, is_active, priority DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_media_items_genres_gin ON media_items USING GIN (genres);
CREATE INDEX IF NOT EXISTS idx_media_items_discovery ON media_items (type, release_year DESC, rating DESC);

INSERT INTO hub_definitions (slug, scope, title_ar, title_en, description_ar, accent, icon, rule, priority)
VALUES
('movies-animation', 'movies', 'أفلام كرتون', 'Animated Movies', 'عالم مرح من أفلام الرسوم المتحركة والمغامرات العائلية.', 'cyan', 'smile', '{"types":["movie"],"tags_any":["كرتون","عائلي","Animation"]}', 100),
('movies-foreign', 'movies', 'أفلام أجنبية', 'Foreign Movies', 'مختارات من أشهر الأفلام العالمية في مكتبتك.', 'violet', 'film', '{"types":["movie"],"tags_any":["أجنبي"]}', 90),
('movies-action', 'movies', 'أفلام أكشن', 'Action Movies', 'مغامرات وإثارة من الأعمال المصنفة أكشن.', 'rose', 'film', '{"types":["movie"],"tags_any":["أكشن","Action"]}', 80),
('series-turkish', 'series', 'مسلسلات تركية', 'Turkish Series', 'حكايات درامية تركية من مختارات المكتبة.', 'amber', 'tv', '{"types":["series"],"tags_any":["تركي"]}', 100),
('series-arabic', 'series', 'مسلسلات عربية', 'Arabic Series', 'دراما عربية وخليجية مختارة.', 'emerald', 'tv', '{"types":["series"],"tags_any":["عربي"]}', 90),
('series-foreign', 'series', 'مسلسلات أجنبية', 'Foreign Series', 'إنتاجات عالمية ومسلسلات أجنبية متنوعة.', 'cyan', 'tv', '{"types":["series"],"tags_any":["أجنبي"]}', 80),
('anime-action', 'anime', 'أنمي أكشن ومغامرة', 'Action Anime', 'رحلات ومعارك من عوالم الأنمي.', 'violet', 'mask', '{"types":["anime"],"tags_any":["أكشن","مغامرة","Action"]}', 100),
('anime-top-rated', 'anime', 'الأعلى تقييماً', 'Top Rated Anime', 'مختارات الأنمي الأعلى تقييماً في المكتبة.', 'rose', 'star', '{"types":["anime"],"rating_gte":7}', 90)
ON CONFLICT (slug) DO NOTHING;
