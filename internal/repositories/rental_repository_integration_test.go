package repositories_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	geom "github.com/twpayne/go-geom"
	pgxgeom "github.com/twpayne/pgx-geom"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// setupTestDB starts a disposable Postgres+PostGIS container, runs the
// project's dbmate migrations against it, and returns a connection pool
// wired up the same way cmd/api/main.go wires the real one.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping testcontainers-backed integration test in -short mode")
	}

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"docker.io/imresamu/postgis:18-3.6-bookworm",
		postgres.WithDatabase("baas"),
		postgres.WithUsername("baas_user"),
		postgres.WithPassword("supersecret"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	dbURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}

	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatalf("parsing connection string: %v", err)
	}
	dbM := dbmate.New(u)
	dbM.MigrationsDir = []string{"../../sql/migrations"}
	if err := dbM.CreateAndMigrate(); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parsing pool config: %v", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxgeom.Register(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("connecting to database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pinging database: %v", err)
	}

	return pool
}

// rectangle builds a simple axis-aligned rectangular polygon covering
// [minX, maxX] x [minY, maxY], satisfying the field/plot "must be a
// rectangle" check constraints.
func rectangle(minX, minY, maxX, maxY float64) *geom.Polygon {
	return geom.NewPolygonFlat(geom.XY, []float64{
		minX, minY,
		maxX, minY,
		maxX, maxY,
		minX, maxY,
		minX, minY,
	}, []int{10})
}

// seedPlot creates a farmer with one field containing one plot, offers the
// first crop in the catalog on that plot, and returns the plot id and that
// crop's id.
func seedPlot(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	farmRepo := repositories.NewFarmRepository(queries)
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)

	farmer, err := accountRepo.CreateFarmer(ctx, models.Account{
		FirstName:    "Old",
		LastName:     "MacDonald",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "irrelevant",
	}, "Green Acres", 76133, "1 Farm Lane", "A small family farm")
	if err != nil {
		t.Fatalf("creating farmer: %v", err)
	}

	farmID, err := farmRepo.GetFarmIDByFarmerID(ctx, farmer.ID)
	if err != nil {
		t.Fatalf("looking up farm: %v", err)
	}

	fieldID, err := fieldRepo.CreateField(ctx, models.Field{
		Name:        "Field 1",
		Farm:        farmID,
		Coordinates: rectangle(0, 0, 10, 10),
	})
	if err != nil {
		t.Fatalf("creating field: %v", err)
	}

	plot, err := plotRepo.CreatePlot(ctx, models.Plot{
		Name:        "Plot 1",
		Field:       fieldID,
		Coordinates: rectangle(1, 1, 5, 5),
	})
	if err != nil {
		t.Fatalf("creating plot: %v", err)
	}

	crop, err := cropRepo.CreateCrop(ctx, "Tomatoes-"+uuid.NewString(), 6)
	if err != nil {
		t.Fatalf("creating crop: %v", err)
	}
	cropID := crop.ID

	if err := cropRepo.SetPlotCrops(ctx, plot.ID, []uuid.UUID{cropID}); err != nil {
		t.Fatalf("offering crop on plot: %v", err)
	}

	return plot.ID, cropID
}

// seedCustomer creates a bare customer account and returns its id.
func seedCustomer(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)

	customer, err := accountRepo.CreateCustomer(ctx, models.Account{
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "irrelevant",
	}, 76133)
	if err != nil {
		t.Fatalf("creating customer: %v", err)
	}
	return customer.ID
}

// TestCreateRental_RejectsOverlappingBooking verifies that the database's
// exclusion constraint (and the repository's mapping of it) stops a plot
// from being rented twice for overlapping periods, even though the service
// layer never checks availability itself before inserting.
func TestCreateRental_RejectsOverlappingBooking(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	plotID, cropID := seedPlot(t, ctx, pool)
	firstCustomer := seedCustomer(t, ctx, pool)
	secondCustomer := seedCustomer(t, ctx, pool)

	rentalRepo := repositories.NewRentalRepository(database.New(pool))

	first, err := rentalRepo.CreateRental(ctx, plotID, firstCustomer, cropID, 6)
	if err != nil {
		t.Fatalf("first booking: unexpected error: %v", err)
	}
	if first.EndAt.Sub(first.StartAt) <= 0 {
		t.Fatalf("expected a positive rental period, got start=%v end=%v", first.StartAt, first.EndAt)
	}

	_, err = rentalRepo.CreateRental(ctx, plotID, secondCustomer, cropID, 6)
	if !errors.Is(err, services.ErrPlotUnavailable) {
		t.Fatalf("second (overlapping) booking: error = %v, want ErrPlotUnavailable", err)
	}

	// Sanity check: only the first booking exists.
	rentals, err := rentalRepo.GetRentalsByCustomer(ctx, firstCustomer)
	if err != nil {
		t.Fatalf("getting rentals: %v", err)
	}
	if len(rentals) != 1 {
		t.Fatalf("customer rentals = %d, want 1", len(rentals))
	}

	otherRentals, err := rentalRepo.GetRentalsByCustomer(ctx, secondCustomer)
	if err != nil {
		t.Fatalf("getting rentals: %v", err)
	}
	if len(otherRentals) != 0 {
		t.Fatalf("second customer rentals = %d, want 0 (booking must have been rejected)", len(otherRentals))
	}
}

// TestGetRentalsByFarm_IncludesHistoricAndScopesToOwnPlots checks the two
// properties a farmer's rental history depends on: an ended rental still
// shows up (unlike GetCustomersOfFarmer, which is active-only), and a rental
// on another farm's plot never leaks in.
func TestGetRentalsByFarm_IncludesHistoricAndScopesToOwnPlots(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)

	_, farmID, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)
	_, otherFarmID, otherPlots, otherCrop := seedFarmerWithPlots(t, ctx, pool, 1)

	activeCustomer := seedCustomer(t, ctx, pool)
	pastCustomer := seedCustomer(t, ctx, pool)
	somebodyElses := seedCustomer(t, ctx, pool)

	active, err := rentalRepo.CreateRental(ctx, plots[0], activeCustomer, cropID, 6)
	if err != nil {
		t.Fatalf("renting plot: %v", err)
	}
	rentPast(t, ctx, pool, plots[1], pastCustomer, cropID)
	if _, err := rentalRepo.CreateRental(ctx, otherPlots[0], somebodyElses, otherCrop, 6); err != nil {
		t.Fatalf("renting other farmer's plot: %v", err)
	}

	rentals, err := rentalRepo.GetRentalsByFarm(ctx, farmID)
	if err != nil {
		t.Fatalf("getting farmer rentals: %v", err)
	}

	if len(rentals) != 2 {
		t.Fatalf("got %d rentals, want 2 (one active, one historic): %+v", len(rentals), rentals)
	}

	byCustomer := make(map[uuid.UUID]models.RentalWithPlotAndCustomer, len(rentals))
	for _, r := range rentals {
		byCustomer[r.Customer.AccountID] = r
	}

	got, ok := byCustomer[activeCustomer]
	if !ok {
		t.Fatalf("active rental missing from results: %+v", rentals)
	}
	if got.ID != active.ID {
		t.Errorf("active rental id = %v, want %v", got.ID, active.ID)
	}
	if got.FieldName != "Field 1" {
		t.Errorf("field name = %q, want %q", got.FieldName, "Field 1")
	}
	if got.Plot.ID != plots[0] {
		t.Errorf("plot id = %v, want %v", got.Plot.ID, plots[0])
	}

	if _, ok := byCustomer[pastCustomer]; !ok {
		t.Errorf("historic (ended) rental missing from results: %+v", rentals)
	}
	if _, ok := byCustomer[somebodyElses]; ok {
		t.Errorf("rental on another farmer's plot leaked into results: %+v", rentals)
	}

	// And the other farmer sees only his own rental.
	others, err := rentalRepo.GetRentalsByFarm(ctx, otherFarmID)
	if err != nil {
		t.Fatalf("getting other farmer's rentals: %v", err)
	}
	if len(others) != 1 || others[0].Customer.AccountID != somebodyElses {
		t.Errorf("other farmer's rentals = %+v, want only the rental by %v", others, somebodyElses)
	}
}
