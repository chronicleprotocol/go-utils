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

package mocks

import (
	"context"

	"github.com/defiweb/go-eth/crypto"
	"github.com/defiweb/go-eth/types"
	"github.com/stretchr/testify/mock"

	"github.com/chronicleprotocol/suite/pkg/core/datapoint"
)

type Recoverer struct {
	mock.Mock
}

func (r *Recoverer) RecoverHash(hash types.Hash, sig types.Signature) (*types.Address, error) {
	args := r.Called(hash, sig)
	return args.Get(0).(*types.Address), args.Error(1)
}

func (r *Recoverer) RecoverMessage(data []byte, sig types.Signature) (*types.Address, error) {
	args := r.Called(data, sig)
	return args.Get(0).(*types.Address), args.Error(1)
}

func (r *Recoverer) RecoverTransaction(tx *types.Transaction) (*types.Address, error) {
	args := r.Called(tx)
	return args.Get(0).(*types.Address), args.Error(1)
}

func (r *Recoverer) Supports(_ context.Context, _ datapoint.Point) bool {
	return true
}

func (r *Recoverer) Recover(_ context.Context, _ string, p datapoint.Point, _ types.Signature) (*types.Address, error) {
	return types.MustAddressFromHexPtr(p.Meta["addr"].(string)), nil
}

var _ crypto.Recoverer = (*Recoverer)(nil)
