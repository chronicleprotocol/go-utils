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

	"github.com/defiweb/go-eth/types"
	"github.com/defiweb/go-eth/wallet"
	"github.com/stretchr/testify/mock"
)

type Key struct {
	mock.Mock
}

func (k *Key) Address() types.Address {
	args := k.Called()
	return args.Get(0).(types.Address)
}

func (k *Key) SignHash(_ context.Context, hash types.Hash) (*types.Signature, error) {
	args := k.Called(hash)
	return args.Get(0).(*types.Signature), args.Error(1)
}

func (k *Key) SignMessage(_ context.Context, data []byte) (*types.Signature, error) {
	args := k.Called(data)
	return args.Get(0).(*types.Signature), args.Error(1)
}

func (k *Key) SignTransaction(_ context.Context, tx *types.Transaction) error {
	args := k.Called(tx)
	return args.Error(0)
}

func (k *Key) VerifyHash(_ context.Context, hash types.Hash, sig types.Signature) bool {
	args := k.Called(hash, sig)
	return args.Bool(0)
}

func (k *Key) VerifyMessage(_ context.Context, data []byte, sig types.Signature) bool {
	args := k.Called(data, sig)
	return args.Bool(0)
}

var _ wallet.Key = (*Key)(nil)
