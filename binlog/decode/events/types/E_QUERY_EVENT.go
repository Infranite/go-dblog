/*
Copyright 2018 liipx(lipengxiang)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"encoding/binary"
	"fmt"

	"github.com/Infranite/go-mysql-binlog/binlog/common"
)

// QueryEvent is the definition of QUERY_EVENT
// https://dev.mysql.com/doc/internals/en/query-event.html
type QueryEvent struct {
	BaseEventBody
	SlaveProxyID     int64
	ExecutionTime    int64
	ErrorCode        uint16
	statusVarsLength int
	StatusVars       []byte
	Status           QueryStatusVars
	Schema           string
	Query            string
}

// QueryStatusVars is the decoded status-vars of QUERY_EVENT
type QueryStatusVars struct {
	Flags2                   uint32
	SQLMode                  uint64
	Catalog                  string
	AutoIncrementIncrement   uint16
	AutoIncrementOffset      uint16
	ClientCharset            uint16
	CollationConnection      uint16
	CollationServer          uint16
	TimeZone                 string
	CatalogNZ                string
	LCTimeNames              uint16
	CharsetDatabase          uint16
	TableMapForUpdate        uint64
	MasterDataWritten        uint32
	InvokerUser              string
	InvokerHost              string
	UpdatedDBNames           []string
	Microseconds             uint32
	UnknownStatusVarsPayload []byte
}

func init() {
	Register(new(QueryEvent))
}

// GetEventType return base env type
func (e *QueryEvent) GetEventType() []uint8 {
	return []uint8{common.QueryEvent}
}

func (e *QueryEvent) Decode(opts ...EventOptionFunc) (EventBody, error) {
	opt := e.InitOption(opts...)
	if opt.Description == nil {
		return nil, fmt.Errorf("invalid binlog version: binary log version info not found")
	}
	if err := requireData(opt.Data, 11); err != nil {
		return nil, err
	}

	var pos int
	event := &QueryEvent{}

	// slave_proxy_id
	event.SlaveProxyID = int64(binary.LittleEndian.Uint32(opt.Data[pos:]))
	pos += 4

	// execution time
	event.ExecutionTime = int64(binary.LittleEndian.Uint32(opt.Data[pos:]))
	pos += 4

	// schema length
	schemaLength := int(opt.Data[pos])
	pos++

	// error-code
	event.ErrorCode = binary.LittleEndian.Uint16(opt.Data[pos:])
	pos += 2

	if opt.Description.BinlogVersion >= 4 {
		if err := requireData(opt.Data[pos:], 2); err != nil {
			return nil, err
		}
		// status-vars length
		event.statusVarsLength = int(binary.LittleEndian.Uint16(opt.Data[pos:]))
		pos += 2

		if err := requireData(opt.Data[pos:], event.statusVarsLength); err != nil {
			return nil, err
		}
		// status-vars
		event.StatusVars = opt.Data[pos : pos+event.statusVarsLength]
		var err error
		event.Status, err = DecodeQueryStatusVars(event.StatusVars)
		if err != nil {
			event.Status = QueryStatusVars{UnknownStatusVarsPayload: event.StatusVars}
		}
		pos += event.statusVarsLength
	}

	if err := requireData(opt.Data[pos:], schemaLength+1); err != nil {
		return nil, err
	}
	// schema
	event.Schema = string(opt.Data[pos : pos+schemaLength])
	pos += schemaLength

	// ignore 0x00
	pos++

	// query
	event.Query = string(opt.Data[pos:])
	return event, nil
}

// Statue will format status_vars of QUERY_EVENT
func (e *QueryEvent) Statue() error {
	status, err := DecodeQueryStatusVars(e.StatusVars)
	if err != nil {
		return err
	}
	e.Status = status
	return nil
}

// DecodeQueryStatusVars decode status-vars payload of QUERY_EVENT
func DecodeQueryStatusVars(data []byte) (QueryStatusVars, error) {
	var status QueryStatusVars
	for i := 0; i < len(data); {
		// got status_vars key
		k := data[i]
		i++

		// decode values
		switch k {
		case common.QFlags2Code:
			if err := requireData(data[i:], 4); err != nil {
				return status, err
			}
			status.Flags2 = binary.LittleEndian.Uint32(data[i:])
			i += 4
		case common.QSQLModeCode:
			if err := requireData(data[i:], 8); err != nil {
				return status, err
			}
			status.SQLMode = binary.LittleEndian.Uint64(data[i:])
			i += 8
		case common.QCatalog:
			if err := requireData(data[i:], 1); err != nil {
				return status, err
			}
			n := int(data[i])
			if err := requireData(data[i+1:], n+1); err != nil {
				return status, err
			}
			status.Catalog = string(data[i+1 : i+1+n])
			i += 1 + n + 1
		case common.QAutoIncrement:
			if err := requireData(data[i:], 4); err != nil {
				return status, err
			}
			status.AutoIncrementIncrement = binary.LittleEndian.Uint16(data[i:])
			status.AutoIncrementOffset = binary.LittleEndian.Uint16(data[i+2:])
			i += 4
		case common.QCharsetCode:
			if err := requireData(data[i:], 6); err != nil {
				return status, err
			}
			status.ClientCharset = binary.LittleEndian.Uint16(data[i:])
			status.CollationConnection = binary.LittleEndian.Uint16(data[i+2:])
			status.CollationServer = binary.LittleEndian.Uint16(data[i+4:])
			i += 6
		case common.QTimeZoneCode:
			if err := requireData(data[i:], 1); err != nil {
				return status, err
			}
			n := int(data[i])
			if err := requireData(data[i+1:], n); err != nil {
				return status, err
			}
			status.TimeZone = string(data[i+1 : i+1+n])
			i += 1 + n
		case common.QCatalogNZCode:
			if err := requireData(data[i:], 1); err != nil {
				return status, err
			}
			n := int(data[i])
			if err := requireData(data[i+1:], n); err != nil {
				return status, err
			}
			status.CatalogNZ = string(data[i+1 : i+1+n])
			i += 1 + n
		case common.QLCTimeNamesCode:
			if err := requireData(data[i:], 2); err != nil {
				return status, err
			}
			status.LCTimeNames = binary.LittleEndian.Uint16(data[i:])
			i += 2
		case common.QCharsetDatabaseCode:
			if err := requireData(data[i:], 2); err != nil {
				return status, err
			}
			status.CharsetDatabase = binary.LittleEndian.Uint16(data[i:])
			i += 2
		case common.QTableMapForUpdateCode:
			if err := requireData(data[i:], 8); err != nil {
				return status, err
			}
			status.TableMapForUpdate = binary.LittleEndian.Uint64(data[i:])
			i += 8
		case common.QMasterDataWrittenCode:
			if err := requireData(data[i:], 4); err != nil {
				return status, err
			}
			status.MasterDataWritten = binary.LittleEndian.Uint32(data[i:])
			i += 4
		case common.QInvokers:
			if err := requireData(data[i:], 1); err != nil {
				return status, err
			}
			userLength := int(data[i])
			i++
			if err := requireData(data[i:], userLength+1); err != nil {
				return status, err
			}
			status.InvokerUser = string(data[i : i+userLength])
			i += userLength
			hostLength := int(data[i])
			i++
			if err := requireData(data[i:], hostLength); err != nil {
				return status, err
			}
			status.InvokerHost = string(data[i : i+hostLength])
			i += hostLength
		case common.QUpdatedDBNames:
			if err := requireData(data[i:], 1); err != nil {
				return status, err
			}
			count := int(data[i])
			i++
			status.UpdatedDBNames = make([]string, 0, count)
			for n := 0; n < count; n++ {
				start := i
				for i < len(data) && data[i] != 0 {
					i++
				}
				if i >= len(data) {
					return status, fmt.Errorf("unterminated updated db name")
				}
				status.UpdatedDBNames = append(status.UpdatedDBNames, string(data[start:i]))
				i++
			}
		case common.QMicroseconds:
			if err := requireData(data[i:], 3); err != nil {
				return status, err
			}
			status.Microseconds = uint32(common.FixedLengthInt(data[i : i+3]))
			i += 3
		default:
			status.UnknownStatusVarsPayload = data[i-1:]
			return status, nil
		}
	}

	return status, nil
}
