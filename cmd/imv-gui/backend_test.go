package main

import (
	"context"
	"errors"
	"testing"
)

func TestCancelScanCancelsActiveScanContext(t *testing.T) {
	backend := NewBackend()
	backend.startup(context.Background())
	ctx, finish := backend.beginScanContext()
	defer finish()

	backend.CancelScan()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("scan context was not canceled: %v", ctx.Err())
	}
}

func TestStartingNewScanCancelsPreviousScanContext(t *testing.T) {
	backend := NewBackend()
	backend.startup(context.Background())
	first, finishFirst := backend.beginScanContext()
	defer finishFirst()
	second, finishSecond := backend.beginScanContext()
	defer finishSecond()

	if !errors.Is(first.Err(), context.Canceled) {
		t.Fatalf("previous scan context was not canceled: %v", first.Err())
	}
	if second.Err() != nil {
		t.Fatalf("new scan context is already canceled: %v", second.Err())
	}
}
