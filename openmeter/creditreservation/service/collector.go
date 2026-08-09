package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger/collector"
)

type settlementCollector interface {
	CollectToAccrued(context.Context, collector.CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error)
	CorrectCollectedAccrued(context.Context, collector.CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error)
}
