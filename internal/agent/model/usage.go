// Purpose: Represent and aggregate provider-reported tokens and optional actual USD charges.
// Pattern: Simple Module.
// Usage: Provider events, agent accounting and terminal snapshots share TokenUsage.
// Related: internal/provider/protocol/chunk.go, internal/agent/run/usage.go.
package model

import (
	"fmt"
	"math"
)

const MaxCallTokens int64 = 1 << 40

type TokenUsage struct {
	Input  int64    `json:"input"`
	Output int64    `json:"output"`
	Cost   *float64 `json:"cost,omitempty"`
}

func (u TokenUsage) Check() error {
	if err := u.CheckTotals(); err != nil {
		return err
	}
	if u.Input > MaxCallTokens || u.Output > MaxCallTokens {
		return fmt.Errorf("invalid per-call token usage")
	}
	return nil
}

// CheckTotals validates accumulated measurements without applying per-call token bounds.
func (u TokenUsage) CheckTotals() error {
	if u.Input < 0 || u.Output < 0 {
		return fmt.Errorf("invalid token usage")
	}
	if u.Cost != nil && (*u.Cost < 0 || math.IsNaN(*u.Cost) || math.IsInf(*u.Cost, 0)) {
		return fmt.Errorf("invalid provider-reported cost")
	}
	return nil
}

// Clone detaches the optional charge from a producer's mutable snapshot.
func (u TokenUsage) Clone() TokenUsage {
	if u.Cost != nil {
		u.Cost = new(*u.Cost)
	}
	return u
}

// Add sums only received measurements; missing cost remains distinct from a reported zero.
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	u.Input += other.Input
	u.Output += other.Output
	if other.Cost != nil {
		cost := *other.Cost
		if u.Cost != nil {
			cost += *u.Cost
		}
		u.Cost = new(cost)
	} else {
		u = u.Clone()
	}
	return u
}
