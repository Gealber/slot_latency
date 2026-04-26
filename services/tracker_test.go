package services

import (
	"testing"
	"time"
)

func TestTrackSlotLogsLatencyWhenGossipArrivesFirst(t *testing.T) {
	var svc TrackerService

	if latency, shouldLog := svc.trackGossip(42, 1_000); shouldLog || latency != 0 {
		t.Fatalf("first gossip event should not log, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	latency, shouldLog := svc.trackYellowstone(42, 2_500)
	if !shouldLog {
		t.Fatal("expected latency to be logged once both timestamps exist")
	}

	if want := 1500 * time.Millisecond; latency != want {
		t.Fatalf("unexpected latency: got %v want %v", latency, want)
	}

	if latency, shouldLog := svc.trackYellowstone(42, 3_000); shouldLog || latency != 0 {
		t.Fatalf("slot should only log once, got latency=%v shouldLog=%v", latency, shouldLog)
	}
}

func TestTrackSlotDropsOutOfOrderEvents(t *testing.T) {
	var svc TrackerService

	if latency, shouldLog := svc.trackYellowstone(42, 1_000); shouldLog || latency != 0 {
		t.Fatalf("first yellowstone event should not log, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	if latency, shouldLog := svc.trackGossip(42, 2_500); shouldLog || latency != 0 {
		t.Fatalf("out-of-order pairing should be dropped, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	if latency, shouldLog := svc.trackYellowstone(42, 3_000); shouldLog || latency != 0 {
		t.Fatalf("dropped slot should stay suppressed, got latency=%v shouldLog=%v", latency, shouldLog)
	}
}

func TestTrackSlotResetsOnRingBufferReuse(t *testing.T) {
	var svc TrackerService

	baseSlot := uint64(10)
	reusedSlot := baseSlot + MAX_SLOTS_TRACK

	if latency, shouldLog := svc.trackGossip(baseSlot, 1_000); shouldLog || latency != 0 {
		t.Fatalf("first gossip event should not log, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	if latency, shouldLog := svc.trackYellowstone(reusedSlot, 2_000); shouldLog || latency != 0 {
		t.Fatalf("reused ring-buffer entry should reset state, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	latency, shouldLog := svc.trackGossip(reusedSlot, 1_500)
	if shouldLog || latency != 0 {
		t.Fatalf("out-of-order reused slot should be dropped, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	if latency, shouldLog := svc.trackGossip(reusedSlot+1, 3_000); shouldLog || latency != 0 {
		t.Fatalf("independent slot should start clean, got latency=%v shouldLog=%v", latency, shouldLog)
	}

	latency, shouldLog = svc.trackYellowstone(reusedSlot+1, 3_900)
	if !shouldLog {
		t.Fatal("expected clean slot after reuse to log")
	}

	if want := 900 * time.Millisecond; latency != want {
		t.Fatalf("unexpected latency after reuse: got %v want %v", latency, want)
	}
}
