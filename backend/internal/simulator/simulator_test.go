package simulator

import (
	"context"
	"moonrider/backend/internal/domain"
	"testing"
	"time"
)

func TestRunReplaysTheSameReadingForDuplicateDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var received []domain.Reading
	err := Run(ctx, "overheat", true, func(r domain.Reading) error {
		received = append(received, r)
		if len(received) == 2 {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received) != 2 {
		t.Fatalf("expected one reading delivered twice; got %d", len(received))
	}
	if received[0] != received[1] {
		t.Fatal("duplicates must preserve the event ID, timestamp, and values")
	}
	if received[0].Temperature <= 80 {
		t.Fatal("overheat scenario must exercise the fault rule")
	}
}
