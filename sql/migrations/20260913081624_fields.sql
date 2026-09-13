-- migrate:up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE field (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    farmer UUID NOT NULL,
    coordinates GEOMETRY(Polygon, 4326) NOT NULL,
    CONSTRAINT coordinates_is_rectangle CHECK (
        ST_Equals(coordinates, ST_Envelope(coordinates))
    ),
    CONSTRAINT fk_field_farmer
        FOREIGN KEY (farmer)
        REFERENCES farmer(account_id)
);

CREATE TABLE plot (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    field UUID NOT NULL,
    coordinates GEOMETRY(Polygon, 4326) NOT NULL,
    CONSTRAINT coordinates_is_rectangle CHECK (
        ST_Equals(coordinates, ST_Envelope(coordinates))
    ),
    CONSTRAINT fk_plot_field
        FOREIGN KEY (field)
        REFERENCES field(id)
);

CREATE FUNCTION check_plot_within_field() RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM field f
        WHERE f.id = NEW.field
        AND ST_Within(NEW.coordinates, f.coordinates)
    ) THEN
        RAISE EXCEPTION 'plot coordinates must be within the boundaries of its field';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_plot_within_field
    BEFORE INSERT OR UPDATE ON plot
    FOR EACH ROW
    EXECUTE FUNCTION check_plot_within_field();

-- migrate:down
DROP TRIGGER trg_plot_within_field ON plot;
DROP FUNCTION check_plot_within_field();
DROP TABLE plot;
DROP TABLE field;
DROP EXTENSION postgis;
