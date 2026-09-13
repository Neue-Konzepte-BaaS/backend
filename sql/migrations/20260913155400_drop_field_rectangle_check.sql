-- migrate:up
-- The "must be a rectangle" CHECK was only ever correct for axis-aligned
-- shapes: it compares the stored geometry to ST_Envelope(geometry) in raw
-- lon/lat degrees, but the frontend's drawing tool (a rotatable rectangle,
-- MapLibre/Terra Draw's TerraDrawAngledRectangleMode) builds shapes in Web
-- Mercator (screen) space, where the projection is angle-preserving. Away
-- from the equator those two spaces disagree: a shape that's a genuine
-- right-angled rectangle on screen can measure a corner as far off as ~66°
-- once converted to raw lon/lat degrees at this project's latitudes. The
-- practical effect: the frontend cannot draw anything BUT a rectangle, yet
-- this constraint rejected a large share of real (rotated) submissions with
-- "field must be a rectangle".
--
-- Dropped outright rather than reworked to a projection-aware check — fields
-- don't need to be constrained to rectangles at all for now.
ALTER TABLE field DROP CONSTRAINT coordinates_is_rectangle;
ALTER TABLE plot DROP CONSTRAINT coordinates_is_rectangle;

-- migrate:down
ALTER TABLE field ADD CONSTRAINT coordinates_is_rectangle CHECK (
    ST_Equals(coordinates, ST_Envelope(coordinates))
);
ALTER TABLE plot ADD CONSTRAINT coordinates_is_rectangle CHECK (
    ST_Equals(coordinates, ST_Envelope(coordinates))
);
