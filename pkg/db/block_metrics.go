package db

/*

This file together with the model, has all the needed methods to interact with the epoch_metrics table of the database

*/

import (
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/spec"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/pkg/errors"
)

// Postgres intregration variables
var (
	UpsertBlock = `
	INSERT INTO t_block_metrics (
		f_timestamp,
		f_epoch, 
		f_slot,
		f_graffiti,
		f_proposer_index,
		f_proposed,
		f_attestations,
		f_deposits,
		f_proposer_slashings,
		f_att_slashings,
		f_voluntary_exits,
		f_sync_bits,
		f_el_fee_recp,
		f_el_gas_limit,
		f_el_gas_used,
		f_el_base_fee_per_gas,
		f_el_block_hash,
		f_el_transactions,
		f_el_block_number,
		f_ssz_size_bytes,
		f_snappy_size_bytes,
		f_compression_time_ms,
		f_decompression_time_ms,
		f_payload_size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		ON CONFLICT ON CONSTRAINT PK_Slot
			DO UPDATE
			SET f_proposed = excluded.f_proposed;
	`
	// update f_proposed is a quick fix that will solve some edgy cases where the drop block
	// reaches before the orphaned block, so it is never dropped
	// with this, the empty block (after removing the orphaned) will conflict
	// and update the proposed.
	// TODO: solve it in next releases with a better task commander
	SelectLastSlot = `
		SELECT f_slot
		FROM t_block_metrics
		ORDER BY f_slot DESC
		LIMIT 1`

	DropBlocksQuery = `
		DELETE FROM t_block_metrics
		WHERE f_slot = $1;
`
)

func insertBlock(inputBlock spec.AgnosticBlock) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)

	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.Timestamp)
	resultArgs = append(resultArgs, inputBlock.Slot/32)
	resultArgs = append(resultArgs, inputBlock.Slot)
	// resultArgs = append(resultArgs, strings.ReplaceAll(string(inputBlock.Graffiti[:]), "\u0000", ""))
	graffiti := strings.ToValidUTF8(string(inputBlock.Graffiti[:]), "?")
	graffiti = strings.ReplaceAll(graffiti, "\u0000", "")
	resultArgs = append(resultArgs, graffiti)
	resultArgs = append(resultArgs, inputBlock.ProposerIndex)
	resultArgs = append(resultArgs, inputBlock.Proposed)
	resultArgs = append(resultArgs, len(inputBlock.Attestations))
	resultArgs = append(resultArgs, len(inputBlock.Deposits))
	resultArgs = append(resultArgs, len(inputBlock.ProposerSlashings))
	resultArgs = append(resultArgs, len(inputBlock.AttesterSlashings))
	resultArgs = append(resultArgs, len(inputBlock.VoluntaryExits))
	resultArgs = append(resultArgs, inputBlock.SyncAggregate.SyncCommitteeBits.Count())
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.FeeRecipient.String())
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.GasLimit)
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.GasUsed)
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.BaseFeeToInt())
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.BlockHash.String())
	resultArgs = append(resultArgs, len(inputBlock.ExecutionPayload.Transactions))
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.BlockNumber)
	resultArgs = append(resultArgs, inputBlock.SSZsize)
	resultArgs = append(resultArgs, inputBlock.SnappySize)
	resultArgs = append(resultArgs, utils.DurationToFloat64Millis(inputBlock.CompressionTime))
	resultArgs = append(resultArgs, utils.DurationToFloat64Millis(inputBlock.DecompressionTime))
	resultArgs = append(resultArgs, inputBlock.ExecutionPayload.PayloadSize)

	return UpsertBlock, resultArgs
}

func BlockOperation(inputBlock spec.AgnosticBlock) (string, []interface{}) {

	q, args := insertBlock(inputBlock)
	return q, args

}

// in case the table did not exist
func (p *PostgresDBService) ObtainLastSlot() (phase0.Slot, error) {
	// create the tables
	rows, err := p.psqlPool.Query(p.ctx, SelectLastSlot)
	if err != nil {
		return 0, errors.Wrap(err, "error obtianing last block from database")
	}
	slot := uint64(0)
	rows.Next()
	rows.Scan(&slot)
	rows.Close()
	return phase0.Slot(slot), nil
}

type BlockDropType phase0.Slot

func (s BlockDropType) Type() spec.ModelType {
	return spec.BlockDropModel
}

func DropBlocks(slot BlockDropType) (string, []interface{}) {
	resultArgs := make([]interface{}, 0)
	resultArgs = append(resultArgs, slot)
	return DropBlocksQuery, resultArgs
}

func (s *PostgresDBService) DeleteBlockMetrics(slot phase0.Slot) {
	s.SingleQuery(DropBlocksQuery, slot)
	s.SingleQuery(DropTransactionsQuery, slot)
}
