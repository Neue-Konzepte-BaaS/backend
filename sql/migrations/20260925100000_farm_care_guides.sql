-- migrate:up
-- A farm's own version of one crop's care guide. The row is the marker that
-- the farm has taken the guide over: while it exists, that farm's tenants read
-- the farm's instructions for the crop and nothing of the default guide; once
-- it is gone, they read the default again. Keeping the marker separate from
-- the instructions is what lets a farm's guide be *empty* — a farmer who
-- deletes every step has an empty guide, not the default back — and its
-- primary key is what makes taking a guide over race-free: two first edits at
-- once insert one marker, and only the one that inserted it copies the default.
CREATE TABLE farm_care_guide (
    farm UUID NOT NULL,
    crop UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (farm, crop),
    CONSTRAINT fk_farm_care_guide_farm
        FOREIGN KEY (farm)
        REFERENCES farm(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_farm_care_guide_crop
        FOREIGN KEY (crop)
        REFERENCES crop(id)
        ON DELETE CASCADE
);

-- farm IS NULL is the default guide an admin maintains; farm = X is farm X's
-- own version. based_on is the default instruction a farm's copy was taken
-- from, which is how a farmer's edit of a default step (the id they were
-- shown) finds the farm's copy of it.
ALTER TABLE care_instruction
    ADD COLUMN farm UUID,
    ADD COLUMN based_on UUID,
    -- A farm's instruction exists only under its marker, so resetting a guide
    -- to the default is deleting one marker row. MATCH SIMPLE (the default)
    -- skips the check for the default guide's NULL farm.
    ADD CONSTRAINT fk_care_instruction_farm_care_guide
        FOREIGN KEY (farm, crop)
        REFERENCES farm_care_guide(farm, crop)
        ON DELETE CASCADE,
    ADD CONSTRAINT fk_care_instruction_based_on
        FOREIGN KEY (based_on)
        REFERENCES care_instruction(id)
        ON DELETE SET NULL;

DROP INDEX idx_care_instruction_crop_week;
CREATE INDEX idx_care_instruction_crop_farm_week ON care_instruction(crop, farm, week);
CREATE INDEX idx_care_instruction_farm_based_on ON care_instruction(farm, based_on) WHERE based_on IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_care_instruction_farm_based_on;
DROP INDEX IF EXISTS idx_care_instruction_crop_farm_week;
DELETE FROM care_instruction WHERE farm IS NOT NULL;
ALTER TABLE care_instruction
    DROP CONSTRAINT fk_care_instruction_based_on,
    DROP CONSTRAINT fk_care_instruction_farm_care_guide,
    DROP COLUMN based_on,
    DROP COLUMN farm;
CREATE INDEX idx_care_instruction_crop_week ON care_instruction(crop, week);
DROP TABLE farm_care_guide;
