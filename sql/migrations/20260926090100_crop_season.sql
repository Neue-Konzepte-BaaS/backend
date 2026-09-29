-- migrate:up
-- season_crop is the rule tying a crop to a season, so renting it is only
-- allowed inside that season's window. farm_id NULL is the default rule an
-- admin sets, applying to every farm that has not overridden it; farm_id set
-- is that farm's own rule, whose season_id must point at a season belonging
-- to that same farm (never another farm's, never the shared default - the
-- database cannot express that cross-row check, so it is enforced in the
-- service layer, the same way farm ownership already is elsewhere). A crop
-- with no row at all, for a farm or as a default, is unrestricted.
CREATE TABLE season_crop (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    season_id UUID NOT NULL,
    crop_id UUID NOT NULL,
    farm_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_season_crop_farm
        FOREIGN KEY (farm_id)
        REFERENCES farm(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_season_crop_season
        FOREIGN KEY (season_id)
        REFERENCES season(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_season_crop_crop
        FOREIGN KEY (crop_id)
        REFERENCES crop(id)
        ON DELETE CASCADE
);

-- At most one rule per crop per farm, and at most one default rule per crop.
-- The COALESCE folds every default row (farm_id IS NULL) onto one sentinel
-- value for uniqueness purposes, since a plain UNIQUE(crop_id, farm_id) would
-- treat every NULL farm_id as distinct and let a crop collect several
-- default rules.
CREATE UNIQUE INDEX idx_season_crop_crop_farm
    ON season_crop (crop_id, COALESCE(farm_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- migrate:down
DROP INDEX IF EXISTS idx_season_crop_crop_farm;
DROP TABLE season_crop;
