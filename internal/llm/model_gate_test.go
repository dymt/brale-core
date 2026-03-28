// 本文件主要内容：验证按 model key 的并发闸门与冷却行为。

package llm

import (
	"context"
	"testing"
	"time"
)

func TestModelGate_SameModelSerialized(t *testing.T) {
	reg := NewModelGateRegistry(1)

	release1, err := reg.Acquire(context.Background(), "m")
	if err != nil {
		t.Fatalf("acquire1: %v", err)
	}
	defer release1()

	acquired := make(chan struct{}, 1)
	release2Ch := make(chan func(), 1)

	go func() {
		release2, err := reg.Acquire(context.Background(), "m")
		if err != nil {
			return
		}
		release2Ch <- release2
		acquired <- struct{}{}
	}()

	select {
	case <-acquired:
		t.Fatalf("expected second acquire to block")
	case <-time.After(50 * time.Millisecond):
	}

	release1()

	select {
	case <-acquired:
		select {
		case release2 := <-release2Ch:
			release2()
		default:
			t.Fatalf("expected release2")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected second acquire after release")
	}
}

func TestModelGate_DifferentModelsParallel(t *testing.T) {
	reg := NewModelGateRegistry(1)

	start := make(chan struct{})
	aDone := make(chan struct{}, 1)
	bDone := make(chan struct{}, 1)

	go func() {
		<-start
		release, err := reg.Acquire(context.Background(), "a")
		if err == nil {
			release()
		}
		aDone <- struct{}{}
	}()
	go func() {
		<-start
		release, err := reg.Acquire(context.Background(), "b")
		if err == nil {
			release()
		}
		bDone <- struct{}{}
	}()

	close(start)

	timeout := time.After(200 * time.Millisecond)
	gotA := false
	gotB := false
	for !gotA || !gotB {
		select {
		case <-aDone:
			gotA = true
		case <-bDone:
			gotB = true
		case <-timeout:
			t.Fatalf("expected both acquires")
		}
	}
}

func TestModelGate_CooldownBlocksAcquire(t *testing.T) {
	reg := NewModelGateRegistry(1)
	wait := 80 * time.Millisecond
	reg.SetCooldown("m", time.Now().Add(wait))

	start := time.Now()
	release, err := reg.Acquire(context.Background(), "m")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	release()

	elapsed := time.Since(start)
	if elapsed < wait/2 {
		t.Fatalf("expected acquire to wait, elapsed=%v", elapsed)
	}
}

func TestModelGate_MinIntervalBlocksAcquire(t *testing.T) {
	reg := NewModelGateRegistry(1)
	wait := 50 * time.Millisecond
	reg.SetMinInterval(wait)

	release, err := reg.Acquire(context.Background(), "m")
	if err != nil {
		t.Fatalf("acquire1: %v", err)
	}
	release()

	start := time.Now()
	release2, err := reg.Acquire(context.Background(), "m")
	if err != nil {
		t.Fatalf("acquire2: %v", err)
	}
	release2()

	elapsed := time.Since(start)
	if elapsed < wait/2 {
		t.Fatalf("expected min interval wait, elapsed=%v", elapsed)
	}
}

func TestNewModelGateRegistry_DefaultLimitMinOne(t *testing.T) {
	cases := []struct {
		name  string
		input int
		want  int
	}{
		{name: "negative", input: -10, want: 1},
		{name: "minus_one", input: -1, want: 1},
		{name: "zero", input: 0, want: 1},
		{name: "one", input: 1, want: 1},
		{name: "two", input: 2, want: 2},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reg := NewModelGateRegistry(tc.input)
			if reg.defaultLimit != tc.want {
				t.Fatalf("defaultLimit=%d, want %d", reg.defaultLimit, tc.want)
			}
		})
	}
}

func TestModelGate_ModelLimitOverride(t *testing.T) {
	reg := NewModelGateRegistry(1)
	reg.SetModelLimits(map[string]int{"m": 2})

	release1, err := reg.Acquire(context.Background(), "m")
	if err != nil {
		t.Fatalf("acquire1: %v", err)
	}
	defer release1()

	release2, err := reg.Acquire(context.Background(), "m")
	if err != nil {
		t.Fatalf("acquire2: %v", err)
	}
	defer release2()

	acquired := make(chan struct{}, 1)
	go func() {
		release3, err := reg.Acquire(context.Background(), "m")
		if err != nil {
			return
		}
		release3()
		acquired <- struct{}{}
	}()

	select {
	case <-acquired:
		t.Fatalf("expected third acquire to block")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestModelGate_ModelLimitOverrideCaseInsensitive(t *testing.T) {
	reg := NewModelGateRegistry(1)
	reg.SetModelLimits(map[string]int{"minimax-m2.5": 2})

	release1, err := reg.Acquire(context.Background(), "MiniMax-M2.5")
	if err != nil {
		t.Fatalf("acquire1: %v", err)
	}
	defer release1()

	release2, err := reg.Acquire(context.Background(), "MiniMax-M2.5")
	if err != nil {
		t.Fatalf("acquire2: %v", err)
	}
	defer release2()

	acquired := make(chan struct{}, 1)
	go func() {
		release3, err := reg.Acquire(context.Background(), "MiniMax-M2.5")
		if err != nil {
			return
		}
		release3()
		acquired <- struct{}{}
	}()

	select {
	case <-acquired:
		t.Fatalf("expected third acquire to block")
	case <-time.After(50 * time.Millisecond):
	}
}
