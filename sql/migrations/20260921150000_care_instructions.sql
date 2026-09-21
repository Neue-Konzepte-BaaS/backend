-- migrate:up
CREATE TABLE care_instruction (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crop UUID NOT NULL,
    -- The week of a *rental*, counted from the day the plot was booked: week 1
    -- is its first seven days, not ISO calendar week 1. Rentals start whenever
    -- the customer books, so a guide pinned to calendar weeks would tell two
    -- tenants of the same crop to do different things on the same day of their
    -- own growing period. The cap is two years — far past any crop's rental
    -- duration, and low enough that a typo cannot store week 100000.
    week INT NOT NULL CHECK (week >= 1 AND week <= 104),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Like announcement, this cascades: an instruction is advice about one
    -- crop and means nothing once that crop leaves the catalog. Note the
    -- asymmetry with rental, whose plain reference is what makes
    -- DeleteCrop answer 409 for a crop someone is renting — a care guide must
    -- not be the thing that blocks an admin from tidying the catalog.
    CONSTRAINT fk_care_instruction_crop
        FOREIGN KEY (crop)
        REFERENCES crop(id)
        ON DELETE CASCADE
);

-- Every read is "the guide for one crop, in week order", so the index carries
-- the sort rather than leaving it to a sort node.
CREATE INDEX idx_care_instruction_crop_week ON care_instruction(crop, week);

-- migrate:down
DROP INDEX IF EXISTS idx_care_instruction_crop_week;
DROP TABLE care_instruction;
