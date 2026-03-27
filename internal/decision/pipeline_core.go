package decision

import (
	"brale-core/internal/market"
	"brale-core/internal/position"
	"brale-core/internal/store"
)

func NewPipelineCoreDeps(st store.Store, positioner *position.PositionService, riskPlans *position.RiskPlanService, priceSource market.PriceSource, states StateProvider) PipelineCoreDeps {
	core := PipelineCoreDeps{
		Store:       st,
		Positioner:  positioner,
		RiskPlans:   riskPlans,
		PriceSource: priceSource,
		States:      states,
	}
	if positioner != nil {
		core.PlanCache = positioner.PlanCache
	}
	return core
}
