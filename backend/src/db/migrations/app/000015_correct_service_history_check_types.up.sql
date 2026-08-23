-- Migration 000014 had to fall back to each service's current check type. Recover
-- the observation-time type only when the historical payload contains a known
-- canonical value. Rows without trustworthy metadata deliberately remain unchanged.
SELECT set_config('app.internal_bypass_rls', 'on', false);

UPDATE app.health_check_results
SET check_type = CASE
    WHEN LOWER(BTRIM(payload ->> 'check_type', E' \t\n\r\f\013')) IN ('http', 'tcp')
        THEN LOWER(BTRIM(payload ->> 'check_type', E' \t\n\r\f\013'))
    WHEN NULLIF(BTRIM(payload ->> 'check_type', E' \t\n\r\f\013'), '') IS NULL
         AND LOWER(BTRIM(payload ->> 'type', E' \t\n\r\f\013')) IN ('http', 'tcp')
        THEN LOWER(BTRIM(payload ->> 'type', E' \t\n\r\f\013'))
    ELSE check_type
END
WHERE LOWER(BTRIM(payload ->> 'check_type', E' \t\n\r\f\013')) IN ('http', 'tcp')
   OR (
       NULLIF(BTRIM(payload ->> 'check_type', E' \t\n\r\f\013'), '') IS NULL
       AND LOWER(BTRIM(payload ->> 'type', E' \t\n\r\f\013')) IN ('http', 'tcp')
   );

SELECT set_config('app.internal_bypass_rls', 'off', false);
