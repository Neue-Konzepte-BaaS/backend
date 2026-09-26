package services

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeCareInstructionRepo is an in-memory care_instruction table with its
// farm_care_guide markers: enough of the real semantics — a farm reads its own
// guide once the marker exists, the default otherwise, and taking a guide over
// copies the default — for the service's copy-on-write rules to be tested
// against something that behaves like the database. err, when set, fails
// every call.
type fakeCareInstructionRepo struct {
	instructions []models.CareInstruction
	basedOn      map[uuid.UUID]uuid.UUID // farm copy id -> default id
	farmGuides   map[models.CropAtFarm]bool
	err          error

	effectiveCalls int
	gotGuides      []models.CropAtFarm
	startCalls     int
}

func newFakeCareRepo(instructions ...models.CareInstruction) *fakeCareInstructionRepo {
	return &fakeCareInstructionRepo{
		instructions: instructions,
		basedOn:      map[uuid.UUID]uuid.UUID{},
		farmGuides:   map[models.CropAtFarm]bool{},
	}
}

func (f *fakeCareInstructionRepo) find(id uuid.UUID) (int, bool) {
	for i, instruction := range f.instructions {
		if instruction.ID == id {
			return i, true
		}
	}
	return 0, false
}

func (f *fakeCareInstructionRepo) CreateCareInstruction(_ context.Context, crop uuid.UUID, farm *uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	if f.err != nil {
		return models.CareInstruction{}, f.err
	}
	if farm != nil && !f.farmGuides[models.CropAtFarm{Crop: crop, Farm: *farm}] {
		return models.CareInstruction{}, ErrNotFound // the marker's foreign key
	}
	instruction := models.CareInstruction{ID: uuid.New(), Crop: crop, Farm: farm, Week: week, Title: title, Body: body}
	f.instructions = append(f.instructions, instruction)
	return instruction, nil
}

func (f *fakeCareInstructionRepo) UpdateCareInstruction(_ context.Context, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	if f.err != nil {
		return models.CareInstruction{}, f.err
	}
	i, ok := f.find(id)
	if !ok {
		return models.CareInstruction{}, ErrNotFound
	}
	f.instructions[i].Week, f.instructions[i].Title, f.instructions[i].Body = week, title, body
	return f.instructions[i], nil
}

func (f *fakeCareInstructionRepo) DeleteCareInstruction(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	i, ok := f.find(id)
	if !ok {
		return ErrNotFound
	}
	f.instructions = append(f.instructions[:i], f.instructions[i+1:]...)
	return nil
}

func (f *fakeCareInstructionRepo) GetCareInstructionByID(_ context.Context, id uuid.UUID) (models.CareInstruction, error) {
	if f.err != nil {
		return models.CareInstruction{}, f.err
	}
	i, ok := f.find(id)
	if !ok {
		return models.CareInstruction{}, ErrNotFound
	}
	return f.instructions[i], nil
}

func (f *fakeCareInstructionRepo) guide(crop uuid.UUID, farm *uuid.UUID) []models.CareInstruction {
	var out []models.CareInstruction
	for _, instruction := range f.instructions {
		if instruction.Crop != crop {
			continue
		}
		if (farm == nil && instruction.Farm == nil) || (farm != nil && instruction.Farm != nil && *instruction.Farm == *farm) {
			out = append(out, instruction)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Week < out[j].Week })
	return out
}

func (f *fakeCareInstructionRepo) GetDefaultCareInstructionsByCrop(_ context.Context, crop uuid.UUID) ([]models.CareInstruction, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.guide(crop, nil), nil
}

func (f *fakeCareInstructionRepo) StartFarmCareGuide(_ context.Context, crop, farm uuid.UUID) error {
	f.startCalls++
	if f.err != nil {
		return f.err
	}
	key := models.CropAtFarm{Crop: crop, Farm: farm}
	if f.farmGuides[key] {
		return nil
	}
	f.farmGuides[key] = true
	for _, instruction := range f.guide(crop, nil) {
		copied := instruction
		copied.ID = uuid.New()
		copied.Farm = &farm
		f.instructions = append(f.instructions, copied)
		f.basedOn[copied.ID] = instruction.ID
	}
	return nil
}

func (f *fakeCareInstructionRepo) GetFarmCopyOfCareInstruction(_ context.Context, farm, basedOn uuid.UUID) (models.CareInstruction, error) {
	if f.err != nil {
		return models.CareInstruction{}, f.err
	}
	for _, instruction := range f.instructions {
		if instruction.Farm != nil && *instruction.Farm == farm && f.basedOn[instruction.ID] == basedOn {
			return instruction, nil
		}
	}
	return models.CareInstruction{}, ErrNotFound
}

func (f *fakeCareInstructionRepo) DeleteFarmCareGuide(_ context.Context, crop, farm uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	key := models.CropAtFarm{Crop: crop, Farm: farm}
	if !f.farmGuides[key] {
		return ErrNotFound
	}
	delete(f.farmGuides, key)
	kept := f.instructions[:0]
	for _, instruction := range f.instructions {
		if instruction.Crop == crop && instruction.Farm != nil && *instruction.Farm == farm {
			continue
		}
		kept = append(kept, instruction)
	}
	f.instructions = kept
	return nil
}

func (f *fakeCareInstructionRepo) HasFarmCareGuide(_ context.Context, crop, farm uuid.UUID) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.farmGuides[models.CropAtFarm{Crop: crop, Farm: farm}], nil
}

func (f *fakeCareInstructionRepo) GetEffectiveCareInstructions(_ context.Context, guides []models.CropAtFarm) (map[models.CropAtFarm][]models.CareInstruction, error) {
	f.effectiveCalls++
	f.gotGuides = guides
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[models.CropAtFarm][]models.CareInstruction, len(guides))
	for _, key := range guides {
		var instructions []models.CareInstruction
		if f.farmGuides[key] {
			instructions = f.guide(key.Crop, &key.Farm)
		} else {
			instructions = f.guide(key.Crop, nil)
		}
		if len(instructions) > 0 {
			out[key] = instructions
		}
	}
	return out, nil
}

// fakeCareRentalRepo only has to answer the one read the care guide makes; the
// embedded interface keeps the other methods off the test's back.
type fakeCareRentalRepo struct {
	RentalRepository

	active []models.ActiveRental
	err    error
}

func (f *fakeCareRentalRepo) GetActiveRentalsByCustomer(context.Context, uuid.UUID) ([]models.ActiveRental, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.active, nil
}

func activeRental(plotName string, crop models.Crop, farm uuid.UUID, currentWeek, totalWeeks int32) models.ActiveRental {
	return models.ActiveRental{
		Rental: models.Rental{
			ID:      uuid.New(),
			PlotID:  uuid.New(),
			CropID:  crop.ID,
			StartAt: time.Now().AddDate(0, 0, -7*int(currentWeek-1)),
			EndAt:   time.Now().AddDate(0, 0, 7*int(totalWeeks-currentWeek+1)),
		},
		PlotName:    plotName,
		FieldName:   "North field",
		FarmID:      farm,
		Crop:        crop,
		CurrentWeek: currentWeek,
		TotalWeeks:  totalWeeks,
	}
}

func instruction(crop uuid.UUID, week int32, title string) models.CareInstruction {
	return models.CareInstruction{ID: uuid.New(), Crop: crop, Week: week, Title: title, Body: "..."}
}

// careFixture is one farmer with a farm, and an admin.
type careFixture struct {
	farmer, farm uuid.UUID
	farmRepo     *fakeFarmRepo
}

func newCareFixture() careFixture {
	farmer, farm := uuid.New(), uuid.New()
	return careFixture{
		farmer:   farmer,
		farm:     farm,
		farmRepo: &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: farm}},
	}
}

func (c careFixture) farmerEditor() CareGuideEditor {
	return CareGuideEditor{AccountID: c.farmer, Role: models.RoleFarmer}
}

var adminEditor = CareGuideEditor{AccountID: uuid.New(), Role: models.RoleAdmin}

func titles(instructions []models.CareInstruction) []string {
	out := make([]string, len(instructions))
	for i, instruction := range instructions {
		out[i] = instruction.Title
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGetCareGuideForCustomer(t *testing.T) {
	tomatoes := models.Crop{ID: uuid.New(), Name: "Tomatoes", DurationMonths: 3}
	beans := models.Crop{ID: uuid.New(), Name: "Beans", DurationMonths: 3}
	farm := uuid.New()

	t.Run("pairs each rented plot with its crop's guide", func(t *testing.T) {
		careRepo := newFakeCareRepo(
			instruction(tomatoes.ID, 1, "Water in"), instruction(tomatoes.ID, 2, "Thin out"),
			instruction(beans.ID, 1, "Set the canes"),
		)
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{
			activeRental("Plot 1", tomatoes, farm, 2, 13),
			activeRental("Plot 2", beans, farm, 2, 13),
		}}
		service := NewCareGuideService(careRepo, rentalRepo, &fakeFarmRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides) != 2 {
			t.Fatalf("got %d guides, want 2", len(guides))
		}
		if guides[0].PlotName != "Plot 1" || len(guides[0].Instructions) != 2 {
			t.Errorf("first guide = %q with %d instructions, want Plot 1 with 2", guides[0].PlotName, len(guides[0].Instructions))
		}
		if guides[1].Crop.Name != "Beans" || len(guides[1].Instructions) != 1 {
			t.Errorf("second guide = %q with %d instructions, want Beans with 1", guides[1].Crop.Name, len(guides[1].Instructions))
		}
		if guides[0].CurrentWeek != 2 || guides[0].TotalWeeks != 13 {
			t.Errorf("week numbers = %d/%d, want 2/13", guides[0].CurrentWeek, guides[0].TotalWeeks)
		}
	})

	t.Run("reads each farm's guide for the plot's own farm", func(t *testing.T) {
		otherFarm := uuid.New()
		careRepo := newFakeCareRepo(instruction(tomatoes.ID, 1, "Default step"))
		if err := careRepo.StartFarmCareGuide(context.Background(), tomatoes.ID, farm); err != nil {
			t.Fatal(err)
		}
		careRepo.instructions = append(careRepo.instructions, models.CareInstruction{ID: uuid.New(), Crop: tomatoes.ID, Farm: &farm, Week: 1, Title: "Farm step"})
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{
			activeRental("On the farm with its own guide", tomatoes, farm, 1, 13),
			activeRental("On a farm reading the default", tomatoes, otherFarm, 1, 13),
		}}
		service := NewCareGuideService(careRepo, rentalRepo, &fakeFarmRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if got, want := titles(guides[0].Instructions), []string{"Default step", "Farm step"}; !equalStrings(got, want) {
			t.Errorf("farm guide = %v, want the farm's copy plus its own step %v", got, want)
		}
		if got, want := titles(guides[1].Instructions), []string{"Default step"}; !equalStrings(got, want) {
			t.Errorf("other farm's guide = %v, want the default %v", got, want)
		}
	})

	t.Run("reads each guide once however many plots follow it", func(t *testing.T) {
		careRepo := newFakeCareRepo(instruction(tomatoes.ID, 1, "Water in"))
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{
			activeRental("Plot 1", tomatoes, farm, 1, 13),
			activeRental("Plot 2", tomatoes, farm, 1, 13),
			activeRental("Plot 3", tomatoes, uuid.New(), 1, 13),
		}}
		service := NewCareGuideService(careRepo, rentalRepo, &fakeFarmRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if careRepo.effectiveCalls != 1 {
			t.Errorf("instruction repo called %d times, want 1", careRepo.effectiveCalls)
		}
		if len(careRepo.gotGuides) != 2 {
			t.Errorf("asked for %d guides, want 2 (one per crop and farm)", len(careRepo.gotGuides))
		}
		for i, guide := range guides {
			if len(guide.Instructions) != 1 {
				t.Errorf("guide %d has %d instructions, want 1", i, len(guide.Instructions))
			}
		}
	})

	t.Run("drops weeks the rental never reaches", func(t *testing.T) {
		careRepo := newFakeCareRepo(
			instruction(tomatoes.ID, 1, "Water in"),
			instruction(tomatoes.ID, 13, "Harvest"),
			instruction(tomatoes.ID, 30, "Never happens"),
		)
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{activeRental("Plot 1", tomatoes, farm, 1, 13)}}
		service := NewCareGuideService(careRepo, rentalRepo, &fakeFarmRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides[0].Instructions) != 2 {
			t.Fatalf("got %d instructions, want 2", len(guides[0].Instructions))
		}
		for _, got := range guides[0].Instructions {
			if got.Week > 13 {
				t.Errorf("instruction for week %d survived a 13-week rental", got.Week)
			}
		}
	})

	t.Run("returns a plot whose crop has no guide yet", func(t *testing.T) {
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{activeRental("Plot 1", tomatoes, farm, 1, 13)}}
		service := NewCareGuideService(newFakeCareRepo(), rentalRepo, &fakeFarmRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides) != 1 {
			t.Fatalf("got %d guides, want 1", len(guides))
		}
		if guides[0].Instructions == nil {
			t.Error("instructions are nil; want an empty slice so the response encodes []")
		}
	})

	t.Run("renting nothing asks for no guides at all", func(t *testing.T) {
		careRepo := newFakeCareRepo()
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, &fakeFarmRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides) != 0 {
			t.Errorf("got %d guides, want 0", len(guides))
		}
		if careRepo.effectiveCalls != 0 {
			t.Errorf("instruction repo called %d times, want 0", careRepo.effectiveCalls)
		}
	})

	t.Run("reports a failing rental read", func(t *testing.T) {
		service := NewCareGuideService(newFakeCareRepo(), &fakeCareRentalRepo{err: errors.New("boom")}, &fakeFarmRepo{})

		if _, err := service.GetCareGuideForCustomer(context.Background(), uuid.New()); err == nil {
			t.Fatal("GetCareGuideForCustomer succeeded, want an error")
		}
	})
}

func TestCareGuideAuthoring(t *testing.T) {
	ctx := context.Background()
	crop := uuid.New()

	t.Run("an admin writes the default guide", func(t *testing.T) {
		c := newCareFixture()
		careRepo := newFakeCareRepo()
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		created, err := service.CreateCareInstruction(ctx, adminEditor, crop, 1, "Water in", "...")
		if err != nil {
			t.Fatalf("CreateCareInstruction: %v", err)
		}
		if created.Farm != nil {
			t.Errorf("admin's instruction has farm %v, want the default guide", *created.Farm)
		}
		if careRepo.startCalls != 0 {
			t.Errorf("admin write took a farm guide over %d times, want 0", careRepo.startCalls)
		}
	})

	t.Run("a farmer's first write copies the default for their farm only", func(t *testing.T) {
		c := newCareFixture()
		careRepo := newFakeCareRepo(instruction(crop, 1, "Water in"), instruction(crop, 2, "Thin out"))
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		created, err := service.CreateCareInstruction(ctx, c.farmerEditor(), crop, 3, "Our own step", "...")
		if err != nil {
			t.Fatalf("CreateCareInstruction: %v", err)
		}
		if created.Farm == nil || *created.Farm != c.farm {
			t.Fatalf("farmer's instruction has farm %v, want %v", created.Farm, c.farm)
		}

		farmGuide, err := service.GetCareInstructionsForCrop(ctx, c.farmerEditor(), crop)
		if err != nil {
			t.Fatalf("GetCareInstructionsForCrop: %v", err)
		}
		if !farmGuide.FarmGuide {
			t.Error("farmer's guide is not marked as the farm's own")
		}
		if got, want := titles(farmGuide.Instructions), []string{"Water in", "Thin out", "Our own step"}; !equalStrings(got, want) {
			t.Errorf("farm guide = %v, want the default plus the new step %v", got, want)
		}

		defaultGuide, err := service.GetCareInstructionsForCrop(ctx, adminEditor, crop)
		if err != nil {
			t.Fatalf("GetCareInstructionsForCrop: %v", err)
		}
		if got, want := titles(defaultGuide.Instructions), []string{"Water in", "Thin out"}; !equalStrings(got, want) {
			t.Errorf("default guide = %v, want it untouched %v", got, want)
		}
	})

	t.Run("a farmer editing a default step edits the farm's copy", func(t *testing.T) {
		c := newCareFixture()
		defaultStep := instruction(crop, 1, "Water in")
		careRepo := newFakeCareRepo(defaultStep)
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		updated, err := service.UpdateCareInstruction(ctx, c.farmerEditor(), defaultStep.ID, 1, "Water in twice", "...")
		if err != nil {
			t.Fatalf("UpdateCareInstruction: %v", err)
		}
		if updated.ID == defaultStep.ID {
			t.Error("the default instruction itself was edited, want the farm's copy")
		}
		if updated.Farm == nil || *updated.Farm != c.farm {
			t.Errorf("edited instruction has farm %v, want %v", updated.Farm, c.farm)
		}
		if got, _ := careRepo.GetCareInstructionByID(ctx, defaultStep.ID); got.Title != "Water in" {
			t.Errorf("default step title = %q, want it unchanged", got.Title)
		}

		// A second edit through the default id lands on the same copy rather
		// than copying again.
		again, err := service.UpdateCareInstruction(ctx, c.farmerEditor(), defaultStep.ID, 1, "Water in thrice", "...")
		if err != nil {
			t.Fatalf("second UpdateCareInstruction: %v", err)
		}
		if again.ID != updated.ID {
			t.Errorf("second edit hit %v, want the same copy %v", again.ID, updated.ID)
		}
	})

	t.Run("a farmer deleting a default step leaves the default alone", func(t *testing.T) {
		c := newCareFixture()
		defaultStep := instruction(crop, 1, "Water in")
		careRepo := newFakeCareRepo(defaultStep)
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		if err := service.DeleteCareInstruction(ctx, c.farmerEditor(), defaultStep.ID); err != nil {
			t.Fatalf("DeleteCareInstruction: %v", err)
		}

		farmGuide, err := service.GetCareInstructionsForCrop(ctx, c.farmerEditor(), crop)
		if err != nil {
			t.Fatalf("GetCareInstructionsForCrop: %v", err)
		}
		if !farmGuide.FarmGuide || len(farmGuide.Instructions) != 0 {
			t.Errorf("farm guide = %v (own: %v), want the farm's own, now empty", titles(farmGuide.Instructions), farmGuide.FarmGuide)
		}
		if _, err := careRepo.GetCareInstructionByID(ctx, defaultStep.ID); err != nil {
			t.Errorf("default step is gone: %v", err)
		}
	})

	t.Run("a farmer cannot touch another farm's instruction", func(t *testing.T) {
		c := newCareFixture()
		otherFarm := uuid.New()
		careRepo := newFakeCareRepo()
		if err := careRepo.StartFarmCareGuide(ctx, crop, otherFarm); err != nil {
			t.Fatal(err)
		}
		theirs, err := careRepo.CreateCareInstruction(ctx, crop, &otherFarm, 1, "Theirs", "...")
		if err != nil {
			t.Fatal(err)
		}
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		if _, err := service.UpdateCareInstruction(ctx, c.farmerEditor(), theirs.ID, 1, "Mine now", "..."); !errors.Is(err, ErrNotFound) {
			t.Errorf("update err = %v, want ErrNotFound", err)
		}
		if err := service.DeleteCareInstruction(ctx, c.farmerEditor(), theirs.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("delete err = %v, want ErrNotFound", err)
		}
	})

	t.Run("an admin may edit a farm's instruction", func(t *testing.T) {
		c := newCareFixture()
		careRepo := newFakeCareRepo()
		if err := careRepo.StartFarmCareGuide(ctx, crop, c.farm); err != nil {
			t.Fatal(err)
		}
		farmStep, err := careRepo.CreateCareInstruction(ctx, crop, &c.farm, 1, "Farm step", "...")
		if err != nil {
			t.Fatal(err)
		}
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		updated, err := service.UpdateCareInstruction(ctx, adminEditor, farmStep.ID, 1, "Moderated", "...")
		if err != nil {
			t.Fatalf("UpdateCareInstruction: %v", err)
		}
		if updated.ID != farmStep.ID || updated.Title != "Moderated" {
			t.Errorf("updated = %+v, want the farm's step edited in place", updated)
		}
	})

	t.Run("reset brings the default back", func(t *testing.T) {
		c := newCareFixture()
		careRepo := newFakeCareRepo(instruction(crop, 1, "Water in"))
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{}, c.farmRepo)

		if _, err := service.CreateCareInstruction(ctx, c.farmerEditor(), crop, 2, "Our own step", "..."); err != nil {
			t.Fatalf("CreateCareInstruction: %v", err)
		}
		if err := service.ResetFarmCareGuide(ctx, c.farmer, crop); err != nil {
			t.Fatalf("ResetFarmCareGuide: %v", err)
		}

		guide, err := service.GetCareInstructionsForCrop(ctx, c.farmerEditor(), crop)
		if err != nil {
			t.Fatalf("GetCareInstructionsForCrop: %v", err)
		}
		if guide.FarmGuide {
			t.Error("guide is still the farm's own after a reset")
		}
		if got, want := titles(guide.Instructions), []string{"Water in"}; !equalStrings(got, want) {
			t.Errorf("guide after reset = %v, want the default %v", got, want)
		}

		if err := service.ResetFarmCareGuide(ctx, c.farmer, crop); !errors.Is(err, ErrNotFound) {
			t.Errorf("second reset err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a farmer without a farm is forbidden", func(t *testing.T) {
		service := NewCareGuideService(newFakeCareRepo(), &fakeCareRentalRepo{}, &fakeFarmRepo{})
		farmless := CareGuideEditor{AccountID: uuid.New(), Role: models.RoleFarmer}

		if _, err := service.CreateCareInstruction(ctx, farmless, crop, 1, "Water in", "..."); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("a customer is never an editor", func(t *testing.T) {
		service := NewCareGuideService(newFakeCareRepo(), &fakeCareRentalRepo{}, &fakeFarmRepo{})
		customer := CareGuideEditor{AccountID: uuid.New(), Role: models.RoleCustomer}

		if _, err := service.GetCareInstructionsForCrop(ctx, customer, crop); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})
}

func TestCareInstructionWritesPassSentinelsThrough(t *testing.T) {
	ctx := context.Background()
	failing := func(err error) CareGuideService {
		repo := newFakeCareRepo()
		repo.err = err
		return NewCareGuideService(repo, &fakeCareRentalRepo{}, &fakeFarmRepo{})
	}

	t.Run("create reports an unknown crop as not found", func(t *testing.T) {
		_, err := failing(ErrNotFound).CreateCareInstruction(ctx, adminEditor, uuid.New(), 1, "Water in", "...")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("update reports an unknown instruction as not found", func(t *testing.T) {
		_, err := failing(ErrNotFound).UpdateCareInstruction(ctx, adminEditor, uuid.New(), 1, "Water in", "...")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete reports an unknown instruction as not found", func(t *testing.T) {
		if err := failing(ErrNotFound).DeleteCareInstruction(ctx, adminEditor, uuid.New()); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a farmer's write for an unknown crop is not found", func(t *testing.T) {
		c := newCareFixture()
		repo := newFakeCareRepo()
		repo.err = ErrNotFound
		service := NewCareGuideService(repo, &fakeCareRentalRepo{}, c.farmRepo)

		if _, err := service.CreateCareInstruction(ctx, c.farmerEditor(), uuid.New(), 1, "Water in", "..."); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a rejected week stays ErrInvalidCareInstruction", func(t *testing.T) {
		_, err := failing(ErrInvalidCareInstruction).CreateCareInstruction(ctx, adminEditor, uuid.New(), 900, "Water in", "...")
		if !errors.Is(err, ErrInvalidCareInstruction) {
			t.Errorf("err = %v, want ErrInvalidCareInstruction", err)
		}
	})

	t.Run("an unexpected repository error is wrapped, not classified", func(t *testing.T) {
		_, err := failing(errors.New("boom")).CreateCareInstruction(ctx, adminEditor, uuid.New(), 1, "Water in", "...")
		if err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want a plain wrapped error", err)
		}
	})
}
