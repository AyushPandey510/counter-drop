-- 0005: one-time setup links so owners and staff choose their own PIN (replaces handing out PINs).
-- A staff row with pin_hash = '' is "pending": it can't sign in until its setup link is used.

CREATE TABLE IF NOT EXISTS cd_setup_links (
    token_hash TEXT PRIMARY KEY,
    staff_id TEXT NOT NULL REFERENCES cd_staff(id) ON DELETE CASCADE,
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK (purpose IN ('setup', 'reset')),
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_cd_setup_links_staff ON cd_setup_links(staff_id);

ALTER TABLE cd_staff
    ADD COLUMN IF NOT EXISTS pin_set_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS removed_at TIMESTAMPTZ;
UPDATE cd_staff SET pin_set_at = created_at WHERE pin_hash <> '' AND pin_set_at IS NULL;

-- Names only need to be unique among current staff, so a removed "Sana" doesn't block a new "Sana".
ALTER TABLE cd_staff DROP CONSTRAINT IF EXISTS cd_staff_shop_id_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_cd_staff_active_name ON cd_staff (shop_id, lower(name)) WHERE active;
