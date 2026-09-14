-- migrate:up
-- Plots are always generated from field geometry — enforcing ST_Within at the
-- DB level only rejects valid rows due to float noise from coordinate projection.
DROP TRIGGER IF EXISTS trg_plot_within_field ON plot;
DROP FUNCTION IF EXISTS check_plot_within_field();

-- migrate:down
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
