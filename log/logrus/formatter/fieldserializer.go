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

package formatter

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/chronicleprotocol/suite/pkg/util/dump"
)

// FieldSerializerFormatter will serialize the log field values to basic types.
// Other types will be serialized to JSON.
type FieldSerializerFormatter struct {
	Formatter         logrus.Formatter
	UseJSONRawMessage bool // If true, then json.RawMessage type will be used for fields serialized to JSON.
}

func (f *FieldSerializerFormatter) Format(e *logrus.Entry) ([]byte, error) {
	data := logrus.Fields{}
	for k, v := range e.Data {
		data[k] = dump.Dump(v)
		if v, ok := data[k].(json.RawMessage); ok && !f.UseJSONRawMessage {
			data[k] = string(v)
		}
	}
	e.Data = data
	return f.Formatter.Format(e)
}
