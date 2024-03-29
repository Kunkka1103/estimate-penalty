package estimate

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
)



// TerminateSector 试图终止一个扇区，并返回估算的罚金和可能出现的错误。
func TerminateSector(ctx context.Context, delegate v0api.FullNode, addr address.Address, sectorNum uint64) (big.Int, error) {

	mi, err := delegate.StateMinerInfo(ctx, addr, types.EmptyTSK)
	if err != nil {
		return big.Zero(), err
	}

	sectorbit := bitfield.New()
	sectorbit.Set(sectorNum)

	loca, err := delegate.StateSectorPartition(ctx, addr, abi.SectorNumber(sectorNum), types.EmptyTSK)
	if err != nil {
		return big.Zero(), fmt.Errorf("get state sector partition %s", err)
	}

	para := miner2.TerminationDeclaration{
		Deadline:  loca.Deadline,
		Partition: loca.Partition,
		Sectors:   sectorbit,
	}
	terminationDeclarationParams := []miner2.TerminationDeclaration{para}

	terminateSectorParams := &miner2.TerminateSectorsParams{
		Terminations: terminationDeclarationParams,
	}

	sp, err := actors.SerializeParams(terminateSectorParams)
	if err != nil {
		return big.Zero(), fmt.Errorf("serializing params: %w", err)
	}

	msg := &types.Message{
		From:   mi.Owner,
		To:     addr,
		Method: builtin.MethodsMiner.TerminateSectors,
		Value:  big.Zero(),
		Params: sp,
	}

	in, err := delegate.StateCall(ctx, msg, types.EmptyTSK)
	if err != nil {
		return big.Zero(), fmt.Errorf("statecall get failed, err: %s", err)
	}

	penalty, found := findPenaltyInInternalExecutions(in.ExecutionTrace.Subcalls)
	if !found {
		return big.Zero(), fmt.Errorf("no penalty found in execution trace")
	}

	return penalty, nil
}

// findPenaltyInInternalExecutions 递归搜索执行跟踪记录，找到与罚金相关的信息。
func findPenaltyInInternalExecutions(trace []types.ExecutionTrace) (big.Int, bool) {
	for _, im := range trace {
		if im.Msg.To.String() == "f099" /*Burn actor*/ {
			return im.Msg.Value, true
		}
		if penalty, found := findPenaltyInInternalExecutions(im.Subcalls); found {
			return penalty, true
		}
	}
	return big.Zero(), false
}
