-- Migration 000014 had to fall back to each service's current check type. Recover
-- the observation-time type only when the historical payload contains a known
-- canonical value. Rows without trustworthy metadata deliberately remain unchanged.
SELECT set_config('app.internal_bypass_rls', 'on', false);

UPDATE app.health_check_results
SET check_type = CASE
    WHEN LOWER(BTRIM(payload ->> 'check_type')) IN ('http', 'tcp')
        THEN LOWER(BTRIM(payload ->> 'check_type'))
    WHEN NULLIF(BTRIM(payload ->> 'check_type'), '') IS NULL
         AND LOWER(BTRIM(payload ->> 'type')) IN ('http', 'tcp')
        THEN LOWER(BTRIM(payload ->> 'type'))
    ELSE check_type
END
WHERE LOWER(BTRIM(payload ->> 'check_type')) IN ('http', 'tcp')
   OR (
       NULLIF(BTRIM(payload ->> 'check_type'), '') IS NULL
       AND LOWER(BTRIM(payload ->> 'type')) IN ('http', 'tcp')
   );

SELECT set_config('app.internal_bypass_rls', 'off', false);
