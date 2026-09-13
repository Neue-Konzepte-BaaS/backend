-- migrate:up
CREATE TABLE crop (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    duration_months INT NOT NULL CHECK (duration_months > 0)
);

CREATE TABLE field_crop (
    field UUID NOT NULL,
    crop UUID NOT NULL,
    PRIMARY KEY (field, crop),
    CONSTRAINT fk_field_crop_field
        FOREIGN KEY (field)
        REFERENCES field(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_field_crop_crop
        FOREIGN KEY (crop)
        REFERENCES crop(id)
);

ALTER TABLE rental ADD COLUMN crop UUID NOT NULL
    CONSTRAINT fk_rental_crop REFERENCES crop(id);

-- migrate:down
ALTER TABLE rental DROP COLUMN crop;
DROP TABLE field_crop;
DROP TABLE crop;
