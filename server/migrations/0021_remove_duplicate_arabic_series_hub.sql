-- Remove the duplicate 'series-arabic' hub in favor of 'series-arabic-drama'
-- Both hubs represent Arabic & Gulf drama series. Keeping 'series-arabic-drama'
-- provides richer branding ("🌟 الدراما العربية والخليجية") and matches the same media items.
DELETE FROM hub_definitions
WHERE slug = 'series-arabic'
  AND EXISTS (
      SELECT 1
      FROM hub_definitions current_hub
      WHERE current_hub.slug = 'series-arabic-drama'
        AND current_hub.is_active = TRUE
  );
