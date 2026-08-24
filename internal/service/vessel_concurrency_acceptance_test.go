package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func vesselFixture(t *testing.T) model.FermentationVessel {
	t.Helper()
	now := time.Now().UTC()
	return model.FermentationVessel{
		VesselCode: "FV-CONC", Name: "Concurrency vessel", WorkingVolumeL: 1000,
		SensorChannels: `["ph","temperature"]`, Location: "Pilot Plant", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
}

func TestUpdateAfterDeactivateRejected(t *testing.T) {
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	svc := NewFermentationVesselService(vesselRepo, repository.NewAuditRepository(db))
	actor := util.Actor{UserID: 1, Username: "scientist", Role: string(constants.RoleProcessScientist), RequestID: "req-007a"}
	vessel := vesselFixture(t)
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Deactivate(context.Background(), vessel.ID, actor); err != nil {
		t.Fatal(err)
	}
	name := "edited-after-deactivate"
	_, err := svc.Update(context.Background(), vessel.ID, dto.UpdateFermentationVesselRequest{Name: &name}, actor)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusConflict {
		t.Fatalf("update of an inactive vessel must return 409 conflict, got: %v", err)
	}
}

type blockingVesselRepo struct {
	repository.FermentationVesselRepository
	mu      sync.Mutex
	blocked bool
	entered chan struct{}
	release chan struct{}
}

func (b *blockingVesselRepo) GetByID(ctx context.Context, id uint) (model.FermentationVessel, error) {
	b.mu.Lock()
	doBlock := !b.blocked
	b.blocked = true
	b.mu.Unlock()
	vessel, err := b.FermentationVesselRepository.GetByID(ctx, id)
	if doBlock {
		close(b.entered)
		<-b.release
	}
	return vessel, err
}

type gateVesselRepo struct {
	repository.FermentationVesselRepository
	entered chan struct{}
	release chan struct{}
}

func (g *gateVesselRepo) GetByID(ctx context.Context, id uint) (model.FermentationVessel, error) {
	vessel, err := g.FermentationVesselRepository.GetByID(ctx, id)
	select {
	case <-g.entered:
	default:
		close(g.entered)
	}
	<-g.release
	return vessel, err
}

func TestConcurrentUpdateDeactivate(t *testing.T) {
	db := newTestDB(t)
	base := repository.NewFermentationVesselRepository(db)
	gate := &gateVesselRepo{
		FermentationVesselRepository: base,
		entered:                      make(chan struct{}),
		release:                      make(chan struct{}),
	}
	svc := NewFermentationVesselService(gate, repository.NewAuditRepository(db))
	actor := util.Actor{UserID: 1, Username: "scientist", Role: string(constants.RoleProcessScientist), RequestID: "req-007b"}
	vessel := vesselFixture(t)
	if err := base.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	deactivateAllowed := make(chan struct{})
	deactivated := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	var updateErr, deactivateErr error
	go func() {
		defer wg.Done()
		<-start
		name := "concurrent-edit"
		_, updateErr = svc.Update(context.Background(), vessel.ID, dto.UpdateFermentationVesselRequest{Name: &name}, actor)
	}()
	go func() {
		defer wg.Done()
		<-start
		<-deactivateAllowed
		_, deactivateErr = base.Deactivate(context.Background(), vessel.ID)
		close(deactivated)
	}()
	close(start)
	<-gate.entered
	close(deactivateAllowed)
	<-deactivated
	close(gate.release)
	wg.Wait()
	if deactivateErr != nil {
		t.Fatalf("deactivate: %v", deactivateErr)
	}
	got, err := svc.Get(context.Background(), vessel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.VesselState != "inactive" {
		t.Fatalf("final state = %s, want inactive", got.VesselState)
	}
	if got.Name == "concurrent-edit" {
		t.Fatalf("vessel was edited after deactivation: name = %q", got.Name)
	}
	var appErr *util.AppError
	if !errors.As(updateErr, &appErr) || appErr.Status != http.StatusConflict {
		t.Fatalf("concurrent update must fail with 409 conflict, got: %v", updateErr)
	}
}
