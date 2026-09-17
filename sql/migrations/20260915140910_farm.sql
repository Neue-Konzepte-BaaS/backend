-- migrate:up
CREATE TABLE farm (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farmer_id UUID NOT NULL UNIQUE,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    founded_at DATE,
    CONSTRAINT fk_farm_farmer
        FOREIGN KEY (farmer_id)
        REFERENCES farmer(account_id)
        ON DELETE CASCADE
);

INSERT INTO farm (farmer_id, name, address)
SELECT account_id, farm_name, '' FROM farmer;

ALTER TABLE field ADD COLUMN farm UUID;
UPDATE field SET farm = farm.id FROM farm WHERE farm.farmer_id = field.farmer;
ALTER TABLE field ALTER COLUMN farm SET NOT NULL;
ALTER TABLE field DROP CONSTRAINT fk_field_farmer;
ALTER TABLE field DROP COLUMN farmer;
ALTER TABLE field ADD CONSTRAINT fk_field_farm FOREIGN KEY (farm) REFERENCES farm(id);

DROP INDEX IF EXISTS idx_field_farmer;
CREATE INDEX idx_field_farm ON field (farm);

ALTER TABLE farmer DROP COLUMN farm_name;

-- migrate:down
ALTER TABLE farmer ADD COLUMN farm_name TEXT;
UPDATE farmer SET farm_name = farm.name FROM farm WHERE farm.farmer_id = farmer.account_id;
ALTER TABLE farmer ALTER COLUMN farm_name SET NOT NULL;

DROP INDEX IF EXISTS idx_field_farm;

ALTER TABLE field ADD COLUMN farmer UUID;
UPDATE field SET farmer = farm.farmer_id FROM farm WHERE farm.id = field.farm;
ALTER TABLE field ALTER COLUMN farmer SET NOT NULL;
ALTER TABLE field DROP CONSTRAINT fk_field_farm;
ALTER TABLE field DROP COLUMN farm;
ALTER TABLE field ADD CONSTRAINT fk_field_farmer FOREIGN KEY (farmer) REFERENCES farmer(account_id);

CREATE INDEX idx_field_farmer ON field (farmer);

DROP TABLE farm;
