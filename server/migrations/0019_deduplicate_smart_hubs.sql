-- Remove the legacy Turkish-series hub duplicated by the expanded editorial hub.
-- Keep the newer, branded definition because both hubs match the same media.
DELETE FROM hub_definitions legacy
WHERE legacy.slug = 'series-turkish'
  AND EXISTS (
      SELECT 1
      FROM hub_definitions current_hub
      WHERE current_hub.slug = 'series-turkish-hits'
        AND current_hub.is_active = TRUE
  );
