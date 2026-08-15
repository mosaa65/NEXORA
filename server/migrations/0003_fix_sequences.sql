-- Fix sequences after explicit ID inserts
SELECT setval(pg_get_serial_sequence('media_items', 'id'), COALESCE((SELECT MAX(id) FROM media_items), 0) + 1, false);
SELECT setval(pg_get_serial_sequence('seasons', 'id'), COALESCE((SELECT MAX(id) FROM seasons), 0) + 1, false);
SELECT setval(pg_get_serial_sequence('video_files', 'id'), COALESCE((SELECT MAX(id) FROM video_files), 0) + 1, false);
SELECT setval(pg_get_serial_sequence('categories', 'id'), COALESCE((SELECT MAX(id) FROM categories), 0) + 1, false);
