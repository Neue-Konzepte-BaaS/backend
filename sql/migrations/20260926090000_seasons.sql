-- migrate:up
-- A season is a recurring yearly calendar window ("Spring: Mar 1 - May 31"),
-- not tied to any particular year - the (month, day) pairs repeat every year.
-- If farm is null, it is a global season that can be referenced, but not modified
-- by every farm. If farm is set, it belongs to this farm and can only be used and
-- modified by this exact farm.
CREATE TABLE season (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farm UUID,
    name TEXT NOT NULL,
    start_month SMALLINT NOT NULL CHECK (start_month BETWEEN 1 AND 12),
    start_day   SMALLINT NOT NULL CHECK (start_day BETWEEN 1 AND 31),
    end_month   SMALLINT NOT NULL CHECK (end_month BETWEEN 1 AND 12),
    end_day     SMALLINT NOT NULL CHECK (end_day BETWEEN 1 AND 31),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_season_farm
        FOREIGN KEY (farm)
        REFERENCES farm(id)
        ON DELETE CASCADE
);

-- Seed the global defaults: a common four-season northern-hemisphere split.
-- Farmers in other regions override these by inserting their own rows here
-- with farm set to their own farm.
INSERT INTO season (farm, name, start_month, start_day, end_month, end_day) VALUES
    (NULL, 'Spring', 3, 1, 5, 31),
    (NULL, 'Summer', 6, 1, 8, 31),
    (NULL, 'Autumn', 9, 1, 11, 30),
    (NULL, 'Winter', 12, 1, 2, 28);

-- migrate:down
DROP TABLE season;
