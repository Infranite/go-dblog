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

package binlog

// https://dev.mysql.com/doc/internals/en/com-query-response.html#packet-Protocol::MYSQL_TYPE_STRING
const (
	MySQLTypeDecimal   = 0x00
	MySQLTypeTiny      = 0x01
	MySQLTypeShort     = 0x02
	MySQLTypeLong      = 0x03
	MySQLTypeFloat     = 0x04
	MySQLTypeDouble    = 0x05
	MySQLTypeNull      = 0x06
	MySQLTypeTimestamp = 0x07
	MySQLTypeLonglong  = 0x08
	MySQLTypeInt24     = 0x09
	MySQLTypeDate      = 0x0a
	MySQLTypeTime      = 0x0b
	MySQLTypeDatetime  = 0x0c
	MySQLTypeYear      = 0x0d
	MySQLTypeNewDate   = 0x0e
	MySQLTypeVarchar   = 0x0f
	MySQLTypeBit       = 0x10

	// mysql 5.6
	MySQLTypeTimestamp2 = 0x11
	MySQLTypeDatetime2  = 0x12
	MySQLTypeTime2      = 0x13
)

const (
	MySQLTypeJson       = 0xf5
	MySQLTypeNewDecimal = 0xf6
	MySQLTypeEnum       = 0xf7
	MySQLTypeSet        = 0xf8
	MySQLTypeTinyBlob   = 0xf9
	MySQLTypeMediumBlob = 0xfa
	MySQLTypeLongBlob   = 0xfb
	MySQLTypeBlob       = 0xfc
	MySQLTypeVarString  = 0xfd
	MySQLTypeString     = 0xfe
	MySQLTypeGeometry   = 0xff
)

// https://dev.mysql.com/doc/internals/en/binlog-event-type.html
const (
	UnknownEvent           = 0x00
	StartEventV3           = 0x01
	QueryEvent             = 0x02
	StopEvent              = 0x03
	RotateEvent            = 0x04
	IntvarEvent            = 0x05
	LoadEvent              = 0x06
	SlaveEvent             = 0x07
	CreateFileEvent        = 0x08
	AppendBlockEvent       = 0x09
	ExecLoadEvent          = 0x0a
	DeleteFileEvent        = 0x0b
	NewLoadEvent           = 0x0c
	RandEvent              = 0x0d
	UserVarEvent           = 0x0e
	FormatDescriptionEvent = 0x0f
	XIDEvent               = 0x10
	BeginLoadQueryEvent    = 0x11
	ExecuteLoadQueryEvent  = 0x12
	TableMapEvent          = 0x13
	WriteRowsEventV0       = 0x14
	UpdateRowsEventV0      = 0x15
	DeleteRowsEventV0      = 0x16
	WriteRowsEventV1       = 0x17
	UpdateRowsEventV1      = 0x18
	DeleteRowsEventV1      = 0x19
	IncidentEvent          = 0x1a
	HeartbeatEvent         = 0x1b
	IgnorableEvent         = 0x1c
	RowsQueryEvent         = 0x1d
	WriteRowsEventV2       = 0x1e
	UpdateRowsEventV2      = 0x1f
	DeleteRowsEventV2      = 0x20
	GTIDEvent              = 0x21
	AnonymousGTIDEvent     = 0x22
	PreviousGTIDEvent      = 0x23
)

// EventType2Str mapping the name of binary log event type
var EventType2Str = map[uint8]string{
	UnknownEvent:           "UNKNOWN_EVENT",
	StartEventV3:           "START_EVENT_V3",
	QueryEvent:             "QUERY_EVENT",
	StopEvent:              "STOP_EVENT",
	RotateEvent:            "ROTATE_EVENT",
	IntvarEvent:            "INTVAR_EVENT",
	LoadEvent:              "LOAD_EVENT",
	SlaveEvent:             "SLAVE_EVENT",
	CreateFileEvent:        "CREATE_FILE_EVENT",
	AppendBlockEvent:       "APPEND_BLOCK_EVENT",
	ExecLoadEvent:          "EXEC_LOAD_EVENT",
	DeleteFileEvent:        "DELETE_FILE_EVENT",
	NewLoadEvent:           "NEW_LOAD_EVENT",
	RandEvent:              "RAND_EVENT",
	UserVarEvent:           "USER_VAR_EVENT",
	FormatDescriptionEvent: "FORMAT_DESCRIPTION_EVENT",
	XIDEvent:               "XID_EVENT",
	BeginLoadQueryEvent:    "BEGIN_LOAD_QUERY_EVENT",
	ExecuteLoadQueryEvent:  "EXECUTE_LOAD_QUERY_EVENT",
	TableMapEvent:          "TABLE_MAP_EVENT",
	WriteRowsEventV0:       "WRITE_ROWS_EVENTv0",
	UpdateRowsEventV0:      "UPDATE_ROWS_EVENTv0",
	DeleteRowsEventV0:      "DELETE_ROWS_EVENTv0",
	WriteRowsEventV1:       "WRITE_ROWS_EVENTv1",
	UpdateRowsEventV1:      "UPDATE_ROWS_EVENTv1",
	DeleteRowsEventV1:      "DELETE_ROWS_EVENTv1",
	IncidentEvent:          "INCIDENT_EVENT",
	HeartbeatEvent:         "HEARTBEAT_EVENT",
	IgnorableEvent:         "IGNORABLE_EVENT",
	RowsQueryEvent:         "ROWS_QUERY_EVENT",
	WriteRowsEventV2:       "WRITE_ROWS_EVENTv2",
	UpdateRowsEventV2:      "UPDATE_ROWS_EVENTv2",
	DeleteRowsEventV2:      "DELETE_ROWS_EVENTv2",
	GTIDEvent:              "GTID_EVENT",
	AnonymousGTIDEvent:     "ANONYMOUS_GTID_EVENT",
	PreviousGTIDEvent:      "PREVIOUS_GTIDS_EVENT",
}

// BINGLOG_CHECKSUM_ALG
const (
	BinlogChecksumAlgOff   byte = 0
	BinlogChecksumAlgCRC32 byte = 1
	BinlogChecksumAlgUndef byte = 255
)

// QUERY_EVENT status_vars
const (
	QFlags2Code            = 0x00
	QSQLModeCode           = 0x01
	QCatalog               = 0x02
	QAutoIncrement         = 0x03
	QCharsetCode           = 0x04
	QTimeZoneCode          = 0x05
	QCatalogNZCode         = 0x06
	QLCTimeNamesCode       = 0x07
	QCharsetDatabaseCode   = 0x08
	QTableMapForUpdateCode = 0x09
	QMasterDataWrittenCode = 0x0a
	QInvokers              = 0x0b
	QUpdatedDBNames        = 0x0c
	QMicroseconds          = 0x0d
)

// QStatusKey2Str is the name of status_vars
var QStatusKey2Str = map[uint8]string{
	QFlags2Code:            "Q_FLAGS2_CODE",
	QSQLModeCode:           "Q_SQL_MODE_CODE",
	QCatalog:               "Q_CATALOG",
	QAutoIncrement:         "Q_AUTO_INCREMENT",
	QCharsetCode:           "Q_CHARSET_CODE",
	QTimeZoneCode:          "Q_TIME_ZONE_CODE",
	QCatalogNZCode:         "Q_CATALOG_NZ_CODE",
	QLCTimeNamesCode:       "Q_LC_TIME_NAMES_CODE",
	QCharsetDatabaseCode:   "Q_CHARSET_DATABASE_CODE",
	QTableMapForUpdateCode: "Q_TABLE_MAP_FOR_UPDATE_CODE",
	QMasterDataWrittenCode: "Q_MASTER_DATA_WRITTEN_CODE",
	QInvokers:              "Q_INVOKERS",
	QUpdatedDBNames:        "Q_UPDATED_DB_NAMES",
	QMicroseconds:          "Q_MICROSECONDS",
}
