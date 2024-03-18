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
	"sync"

	"github.com/stretchr/testify/mock"

	"github.com/chronicleprotocol/go-utils/log"
)

type Logger struct {
	mock mock.Mock
	mu   sync.Mutex

	fields log.Fields
}

func New() *Logger {
	return &Logger{}
}

func (l *Logger) Mock() *mock.Mock {
	l.mu.Lock()
	defer l.mu.Unlock()
	return &l.mock
}

func (l *Logger) Level() log.Level {
	return log.Debug
}

func (l *Logger) WithField(key string, value any) log.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	args := l.mock.Called(key, value)
	return args.Get(0).(log.Logger)
}

func (l *Logger) WithFields(fields log.Fields) log.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	args := l.mock.Called(fields)
	return args.Get(0).(log.Logger)
}

func (l *Logger) WithError(err error) log.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	args := l.mock.Called(err)
	return args.Get(0).(log.Logger)
}

func (l *Logger) WithAdvice(advice string) log.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	args := l.mock.Called(advice)
	return args.Get(0).(log.Logger)
}

func (l *Logger) Debug(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mock.Called(args)
}

func (l *Logger) Info(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mock.Called(args)
}

func (l *Logger) Warn(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mock.Called(args)
}

func (l *Logger) Error(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mock.Called(args)
}

func (l *Logger) Panic(args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mock.Called(args)
}
