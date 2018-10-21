# go-mysql-binlog
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/liipx/go-mysql-binlog/blob/master/LICENSE) 
[![Go Report Card](https://goreportcard.com/badge/github.com/liipx/go-mysql-binlog)](https://goreportcard.com/report/github.com/liipx/go-mysql-binlog)


MySQL binary log analyzer in Golang.

[中文说明](https://github.com/liipx/go-mysql-binlog/blob/master/doc/doc_zh.md)

## Example
```go
func main() {
	decoder, err := binlog.NewBinFileDecoder("./testdata/mysql-bin.000004")
	if err != nil {
		panic(err)
		return
	}
    
	num := 0
	maxEventCount := 0
	err = decoder.WalkEvent(func(event *binlog.BinEvent) (isContinue bool, err error) {
		fmt.Printf("Got %s: \n\t", binlog.EventType2Str[event.Header.EventType])
		fmt.Println(event.Header)
		
		// show details if you need
		// if event.Body != nil {
		// 	pretty.Println(event.Body)
		// }
		//
		
		fmt.Println(strings.Repeat("=", 100))
		count ++
		return maxEventCount > num || maxEventCount == 0, nil
	}, nil)
    
	if err != nil {
		panic(err)
	}

}
```
### Output:
```text
Got FORMAT_DESCRIPTION_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:119, LogPos:123, Flag:0x1
====================================================================================================
Got PREVIOUS_GTIDS_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:31, LogPos:154, Flag:0x80
====================================================================================================
Got ANONYMOUS_GTID_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:65, LogPos:219, Flag:0x0
====================================================================================================
Got QUERY_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:79, LogPos:298, Flag:0x8
====================================================================================================
Got TABLE_MAP_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:64, LogPos:362, Flag:0x0
====================================================================================================
Got WRITE_ROWS_EVENTv2: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:197, LogPos:559, Flag:0x0
====================================================================================================
Got XID_EVENT: 
	Time:2018-09-22 18:24:30 +0800 CST, ServerID:1537611870, EventSize:31, LogPos:590, Flag:0x0
====================================================================================================

```

## Progress
|EventType|Supported|
|---|---|
|UNKNOWN_EVENT|✔|
|START_EVENT_V3||
|QUERY_EVENT|✔|
|STOP_EVENT||
|ROTATE_EVENT|✔|
|INTVAR_EVENT|✔|
|LOAD_EVENT||
|SLAVE_EVENT||
|CREATE_FILE_EVENT||
|APPEND_BLOCK_EVENT||
|EXEC_LOAD_EVENT||
|DELETE_FILE_EVENT||
|NEW_LOAD_EVENT||
|RAND_EVENT||
|USER_VAR_EVENT||
|FORMAT_DESCRIPTION_EVENT|✔|
|XID_EVENT|✔|
|BEGIN_LOAD_QUERY_EVENT||
|EXECUTE_LOAD_QUERY_EVENT||
|TABLE_MAP_EVENT|✔|
|WRITE_ROWS_EVENTv0||
|UPDATE_ROWS_EVENTv0||
|DELETE_ROWS_EVENTv0||
|WRITE_ROWS_EVENTv1||
|UPDATE_ROWS_EVENTv1||
|DELETE_ROWS_EVENTv1||
|INCIDENT_EVENT||
|HEARTBEAT_EVENT||
|IGNORABLE_EVENT||
|ROWS_QUERY_EVENT||
|WRITE_ROWS_EVENTv2||
|UPDATE_ROWS_EVENTv2||
|DELETE_ROWS_EVENTv2||
|GTID_EVENT|✔|
|ANONYMOUS_GTID_EVENT|✔|
|PREVIOUS_GTIDS_EVENT|✔|

## TODO
1. Support all mysql binlog event.
1. Get binlog event through network connections.
1. Multi threads binlog dumper.
1. Flash back base on row format binary log.
1. more.