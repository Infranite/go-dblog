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

	"github.com/liipx/go-mysql-binlog/binlog/common"
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
	Schema           string
	Query            string
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
		// status-vars length
		event.statusVarsLength = int(binary.LittleEndian.Uint16(opt.Data[pos:]))
		pos += 2

		// status-vars
		event.StatusVars = opt.Data[pos : pos+event.statusVarsLength]
		pos += event.statusVarsLength
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
// TODO decode QUERY_EVENT status_var
func (e *QueryEvent) Statue() error {
	fmt.Println(e.statusVarsLength)
	for i := 0; i < e.statusVarsLength; {
		// got status_vars key
		k := e.StatusVars[i]
		i++

		// decode values
		switch k {
		case common.QFlags2Code:
			v := e.StatusVars[i : i+4]
			i += 4

			// TODO
			fmt.Println("QFlags2Code", v)
		case common.QSQLModeCode:
			v := e.StatusVars[i : i+8]
			i += 8

			// TODO
			fmt.Println("QSQLModeCode", v)
		case common.QCatalog:
			n := int(e.StatusVars[i])
			v := string(e.StatusVars[i+1 : i+1+n])
			i += 1 + n + 1

			// TODO
			fmt.Println("QCatalog", v)
		case common.QAutoIncrement:
			increment := binary.LittleEndian.Uint32(e.StatusVars[i:])
			offset := binary.LittleEndian.Uint32(e.StatusVars[i+2:])

			// TODO
			fmt.Printf("QAutoIncrement %d, %d\n", increment, offset)
		case common.QCharsetCode:
			clientCharSet := e.StatusVars[i : i+2]
			i += 2
			collationConnection := e.StatusVars[i : i+2]
			i += 2
			collationServer := e.StatusVars[i : i+2]
			i += 2

			// TODO
			fmt.Println("QCharsetCode", clientCharSet, collationConnection, collationServer)
		case common.QTimeZoneCode:
			n := int(e.StatusVars[i])
			v := string(e.StatusVars[i+1 : i+1+n])
			i += 1 + n

			// TODO
			fmt.Println("QTimeZoneCode", v)
		case common.QCatalogNZCode:
			n := int(e.StatusVars[i])
			v := string(e.StatusVars[i+1 : i+1+n])
			i += 1 + n

			// TODO
			fmt.Printf("QCatalogNZCode %s\n", v)
		case common.QLCTimeNamesCode:
			// TODO
			fmt.Println("QLCTimeNamesCode")
		case common.QCharsetDatabaseCode:
			// TODO
			fmt.Println("QCharsetDatabaseCode")
		case common.QTableMapForUpdateCode:
			// TODO
			fmt.Println("QTableMapForUpdateCode")
		case common.QMasterDataWrittenCode:
			// TODO
			fmt.Println("QMasterDataWrittenCode")
		case common.QInvokers:
			// TODO
			fmt.Println("QInvokers")
		case common.QUpdatedDBNames:
			// TODO
			fmt.Println("QUpdatedDBNames")
		case common.QMicroseconds:
			microseconds := binary.LittleEndian.Uint32(e.StatusVars[i:])
			i += 3

			// TODO
			fmt.Printf("QMicroseconds %d\n", microseconds)
		default:
			return fmt.Errorf("unknown status var %x", k)
		}
	}

	return nil
}
