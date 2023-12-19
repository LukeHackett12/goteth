package db

import (
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/migalabs/goteth/pkg/spec"
)

// Postgres intregration variables
var (
	valRewardsTable      = "t_validator_rewards_summary"
	insertValidatorQuery = `
	INSERT INTO %s (	
		f_val_idx, 
		f_epoch, 
		f_balance_eth, 
		f_reward, 
		f_max_reward,
		f_max_att_reward,
		f_max_sync_reward,
		f_att_slot,
		f_base_reward,
		f_in_sync_committee,
		f_missing_source,
		f_missing_target,
		f_missing_head,
		f_status,
		f_block_api_reward,
		f_block_experimental_reward) VALUES`

	deleteValidatorRewardsInEpochQuery = `
		DELETE FROM %s
		WHERE f_epoch = $1;
	`
)

type InsertValidators struct {
	vals []spec.ValidatorRewards
}

func (d InsertValidators) Table() string {
	return valRewardsTable
}

func (d *InsertValidators) Append(newVal spec.ValidatorRewards) {
	d.vals = append(d.vals, newVal)
}

func (d InsertValidators) Columns() int {
	return len(d.Input().Columns())
}

func (d InsertValidators) Rows() int {
	return len(d.vals)
}

func (d InsertValidators) Query() string {
	return fmt.Sprintf(insertValidatorQuery, valRewardsTable)
}
func (d InsertValidators) Input() proto.Input {
	// one object per column
	var (
		f_val_idx                   proto.ColUInt64
		f_epoch                     proto.ColUInt64
		f_balance_eth               proto.ColFloat32
		f_reward                    proto.ColInt64
		f_max_reward                proto.ColUInt64
		f_max_att_reward            proto.ColUInt64
		f_max_sync_reward           proto.ColUInt64
		f_att_slot                  proto.ColUInt64
		f_base_reward               proto.ColUInt64
		f_in_sync_committee         proto.ColBool
		f_missing_source            proto.ColBool
		f_missing_target            proto.ColBool
		f_missing_head              proto.ColBool
		f_status                    proto.ColUInt8
		f_block_api_reward          proto.ColUInt64
		f_block_experimental_reward proto.ColUInt64
	)

	for _, val := range d.vals {
		f_val_idx.Append(uint64(val.ValidatorIndex))
		f_epoch.Append(uint64(val.Epoch))
		f_balance_eth.Append(float32(val.BalanceToEth()))
		f_reward.Append(int64(val.Reward))
		f_max_reward.Append(uint64(val.MaxReward))
		f_max_att_reward.Append(uint64(val.AttestationReward))
		f_max_sync_reward.Append(uint64(val.SyncCommitteeReward))
		f_att_slot.Append(uint64(val.AttSlot))
		f_base_reward.Append(uint64(val.BaseReward))
		f_in_sync_committee.Append(val.InSyncCommittee)
		f_missing_source.Append(val.MissingSource)
		f_missing_target.Append(val.MissingTarget)
		f_missing_head.Append(val.MissingHead)
		f_status.Append(uint8(val.Status))
		f_block_api_reward.Append(uint64(val.ProposerApiReward))
		f_block_experimental_reward.Append(uint64(val.ProposerManualReward))
	}

	return proto.Input{
		{Name: "f_val_idx", Data: f_val_idx},
		{Name: "f_epoch", Data: f_epoch},
		{Name: "f_balance_eth", Data: f_balance_eth},
		{Name: "f_reward", Data: f_reward},
		{Name: "f_max_reward", Data: f_max_reward},
		{Name: "f_max_att_reward", Data: f_max_att_reward},
		{Name: "f_max_sync_reward", Data: f_max_sync_reward},
		{Name: "f_att_slot", Data: f_att_slot},
		{Name: "f_base_reward", Data: f_base_reward},
		{Name: "f_in_sync_committee", Data: f_in_sync_committee},
		{Name: "f_missing_source", Data: f_missing_source},
		{Name: "f_missing_target", Data: f_missing_target},
		{Name: "f_missing_head", Data: f_missing_head},
		{Name: "f_status", Data: f_status},
		{Name: "f_block_api_reward", Data: f_block_api_reward},
		{Name: "f_block_experimental_reward", Data: f_block_experimental_reward},
	}
}

type DeleteValRewards struct {
	epoch phase0.Epoch
}

func (d DeleteValRewards) Query() string {
	return fmt.Sprintf(deleteValidatorRewardsInEpochQuery, valRewardsTable)
}

func (d DeleteValRewards) Table() string {
	return valRewardsTable
}

func (d DeleteValRewards) Args() []any {
	return []any{d.epoch}
}
