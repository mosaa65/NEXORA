-- Add master categories to categories table
INSERT INTO categories (name_ar, name_en, slug) VALUES
    ('رمضانيات', 'Ramadan', 'ramadan'),
    ('مصارعة ورياضة', 'Wrestling & Sports', 'wrestling'),
    ('موسيقى وصوتيات', 'Music & Audio', 'music'),
    ('برامج وتطبيقات', 'Apps & Shows', 'apps')
ON CONFLICT (slug) DO NOTHING;
