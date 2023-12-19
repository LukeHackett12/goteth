package analyzer

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/spec/metrics"
)

var (
	epochProcesserTag = "epoch="
)

// We always provide the epoch we transition to
// To process the transition from epoch 9 to 10, we provide 10 and we retrieve 8, 9, 10
func (s *ChainAnalyzer) ProcessStateTransitionMetrics(epoch phase0.Epoch) {

	if !s.metrics.Epoch {
		return
	}

	routineKey := fmt.Sprintf("%s%d", epochProcesserTag, epoch)
	s.processerBook.Acquire(routineKey) // resgiter we are about to process metrics for epoch

	// Retrieve states to process metrics

	prevState := &spec.AgnosticState{}
	currentState := &spec.AgnosticState{}
	nextState := &spec.AgnosticState{}

	// this state may never be downloaded if it is below initSlot
	if epoch >= 2 && epoch-2 >= phase0.Epoch(s.initSlot/spec.SlotsPerEpoch) {
		prevState = s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch) - 2)
	}
	if epoch >= 1 && epoch-1 >= phase0.Epoch(s.initSlot/spec.SlotsPerEpoch) {
		currentState = s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch) - 1)
	}
	nextState = s.downloadCache.StateHistory.Wait(EpochTo[uint64](epoch))

	bundle, err := metrics.StateMetricsByForkVersion(nextState, currentState, prevState, s.cli.Api)
	if err != nil {
		s.processerBook.FreePage(routineKey)
		log.Errorf("could not parse bundle metrics at epoch: %s", err)
		s.stop = true
	}

	emptyRoot := phase0.Root{}

	// If nextState is filled, we can process proposer duties
	if nextState.StateRoot != emptyRoot {
		s.processEpochDuties(bundle)

		s.processValLastStatus(bundle)

		// If currentState and nextState are filled, we can process epoch metrics
		if currentState.StateRoot != emptyRoot {
			s.processPoolMetrics(bundle.GetMetricsBase().CurrentState.Epoch)
			s.processEpochMetrics(bundle)

			// If prevState, currentState and nextState are filled, we can process validator rewards
			if prevState.StateRoot != emptyRoot && s.metrics.ValidatorRewards {
				s.processEpochValRewards(bundle)
			}
		}
	}

	s.processerBook.FreePage(routineKey)

}

func (s *ChainAnalyzer) processEpochMetrics(bundle metrics.StateMetrics) {

	var epochs db.InsertEpochs

	// we need sameEpoch and nextEpoch

	epoch := bundle.GetMetricsBase().ExportToEpoch()
	epochs.Append(epoch)

	log.Debugf("persisting epoch metrics: epoch %d", epoch.Epoch)

	err := s.dbClient.Persist(epochs)
	if err != nil {
		log.Fatalf("error persisting epoch: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processPoolMetrics(epoch phase0.Epoch) {

	log.Debugf("persisting pool summaries: epoch %d", epoch)

	err := s.dbClient.InsertPoolSummary(epoch)

	// we need sameEpoch and nextEpoch

	if err != nil {
		log.Fatalf("error persisting pool metrics: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processEpochDuties(bundle metrics.StateMetrics) {

	missedBlocks := bundle.GetMetricsBase().NextState.MissedBlocks

	var duties db.InsertProposerDuties

	for _, item := range bundle.GetMetricsBase().NextState.EpochStructs.ProposerDuties {

		newDuty := spec.ProposerDuty{
			ValIdx:       item.ValidatorIndex,
			ProposerSlot: item.Slot,
			Proposed:     true,
		}
		for _, item := range missedBlocks {
			if newDuty.ProposerSlot == item { // we found the proposer slot in the missed blocks
				newDuty.Proposed = false
			}
		}
		duties.Append(newDuty)
	}

	err := s.dbClient.Persist(duties)
	if err != nil {
		log.Fatalf("error persisting proposer duties: %s", err.Error())
	}

}

func (s *ChainAnalyzer) processValLastStatus(bundle metrics.StateMetrics) {

	if s.downloadMode == "finalized" {
		var valStatusArr db.InsertValidatorLastStatuses
		for valIdx, validator := range bundle.GetMetricsBase().NextState.Validators {

			newVal := spec.ValidatorLastStatus{
				ValIdx:          phase0.ValidatorIndex(valIdx),
				Epoch:           bundle.GetMetricsBase().NextState.Epoch,
				CurrentBalance:  bundle.GetMetricsBase().NextState.Balances[valIdx],
				CurrentStatus:   bundle.GetMetricsBase().NextState.GetValStatus(phase0.ValidatorIndex(valIdx)),
				Slashed:         validator.Slashed,
				ActivationEpoch: validator.ActivationEpoch,
				WithdrawalEpoch: validator.WithdrawableEpoch,
				ExitEpoch:       validator.ExitEpoch,
				PublicKey:       validator.PublicKey,
			}
			valStatusArr.Append(newVal)
		}
		if valStatusArr.Rows() > 0 { // persist everything

			err := s.dbClient.Persist(valStatusArr)
			if err != nil {
				log.Fatalf("error persisting val_status: %s", err.Error())
			}
			err = s.dbClient.Delete(db.DeleteValLastStatus{Epoch: bundle.GetMetricsBase().NextState.Epoch})
			if err != nil {
				log.Fatalf("error deleting val_status: %s", err.Error())
			}
		}
	}
}

func (s *ChainAnalyzer) processEpochValRewards(bundle metrics.StateMetrics) {

	if s.metrics.ValidatorRewards { // only if flag is activated
		var insertValsObj db.InsertValidators
		log.Debugf("persising validator metrics: epoch %d", bundle.GetMetricsBase().NextState.Epoch)

		// process each validator
		for valIdx := range bundle.GetMetricsBase().NextState.Validators {

			if valIdx >= len(bundle.GetMetricsBase().NextState.Validators) {
				continue // validator is not in the chain yet
			}
			// get max reward at given epoch using the formulas
			maxRewards, err := bundle.GetMaxReward(phase0.ValidatorIndex(valIdx))

			if err != nil {
				log.Errorf("Error obtaining max reward: %s", err.Error())
				continue
			}

			insertValsObj.Append(maxRewards)
		}
		if insertValsObj.Rows() > 0 { // persist everything
			err := s.dbClient.Persist(insertValsObj)
			if err != nil {
				log.Fatalf("error persisting val_rewards: %s", err.Error())
			}

		}

	}
}
