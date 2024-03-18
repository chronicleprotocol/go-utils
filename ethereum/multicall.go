//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ethereum

import (
	"context"
	"fmt"

	"github.com/defiweb/go-eth/abi"
	"github.com/defiweb/go-eth/rpc"
	"github.com/defiweb/go-eth/types"

	"github.com/chronicleprotocol/suite/pkg/core/contract/multicall"
)

// MultiCall calls multiple contracts in a single call.
//
// https://github.com/mds1/multicall/
//
// Deprecated: Use the multicall.AggregateCallables instead.
func MultiCall(
	ctx context.Context,
	client rpc.RPC,
	calls []types.Call,
	blockNumber types.BlockNumber,
) ([][]byte, error) {
	type multicallCall struct {
		Target types.Address `abi:"target"`
		Data   []byte        `abi:"callData"`
	}
	var (
		multicallCalls   []multicallCall
		multicallResults [][]byte
	)
	for _, call := range calls {
		if call.To == nil {
			return nil, fmt.Errorf("multicall: call to nil address")
		}
		multicallCalls = append(multicallCalls, multicallCall{
			Target: *call.To,
			Data:   call.Input,
		})
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("multicall: getting chain id failed: %w", err)
	}
	callata, err := multicallMethod.EncodeArgs(multicallCalls)
	if err != nil {
		return nil, fmt.Errorf("multicall: encoding arguments failed: %w", err)
	}
	addr := multicall.Address(chainID)
	resp, _, err := client.Call(ctx, &types.Call{
		To:    &addr,
		Input: callata,
	}, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("multicall: call failed: %w", err)
	}
	if err := multicallMethod.DecodeValues(resp, nil, &multicallResults); err != nil {
		return nil, fmt.Errorf("multicall: decoding results failed: %w", err)
	}
	if len(calls) != len(multicallResults) {
		return nil, fmt.Errorf("unexpected number of multicall results, expected %d, got %d",
			len(calls), len(multicallResults))
	}
	return multicallResults, nil
}

var multicallMethod = abi.MustParseMethod(`
	function aggregate(
		(address target, bytes callData)[] memory calls
	) public returns (
		uint256 blockNumber, 
		bytes[] memory returnData
	)`,
)
