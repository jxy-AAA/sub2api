//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCleanupRegistryRunReturnsWhenParallelStepBlocks(t *testing.T) {
	registry := &CleanupRegistry{}
	blocked := make(chan struct{})
	sequentialRan := false

	registry.AddParallel("blocked", func() error {
		<-blocked
		return nil
	})
	registry.AddSequential("sequential", func() error {
		sequentialRan = true
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		registry.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cleanup registry did not return after context timeout")
	}

	close(blocked)
	require.False(t, sequentialRan)
}

func TestCleanupRegistryRunReturnsWhenSequentialStepBlocks(t *testing.T) {
	registry := &CleanupRegistry{}
	blocked := make(chan struct{})
	secondStepRan := false

	registry.AddSequential("blocked", func() error {
		<-blocked
		return nil
	})
	registry.AddSequential("after-blocked", func() error {
		secondStepRan = true
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		registry.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cleanup registry did not return after sequential timeout")
	}

	close(blocked)
	require.False(t, secondStepRan)
}
