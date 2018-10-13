# go-mysql-binlog
MySQL/MariaDB binary log analyzer in Golang.

[中文说明](https://github.com/liipx/go-mysql-binlog/doc/doc_zh.md)

## Example
```go
func main() {
	decoder, err := binlog.NewBinFileDecoder("/your/log/path/mysql-bin.000001")
	if err != nil {
		t.Error(err)
		return
	}

	err = decoder.WalkEvent(func(event *binlog.BinEvent) (isContinue bool, err error) {
		pretty.Println(event)
		return true, nil
	}, nil)

	if err != nil {
		t.Error(err)
	}

}
```
### Output:
```text
&binlog.BinEvent{
    Header: &binlog.BinEventHeader{Timestamp:1536368222, EventType:0xf, ServerID:11111, EventSize:103, LogPos:107, Flag:0x0},
    Body:   &binlog.BinFmtDescEvent{
        BinlogVersion:         4,
        MySQLVersion:          "5.5.31-log",
        CreateTime:            0,
        EventHeaderLength:     19,
        EventTypeHeaderLength: {0x38, 0xd, 0x0, 0x8, 0x0, 0x12, 0x0, 0x4, 0x4, 0x4, 0x4, 0x12, 0x0, 0x0, 0x54, 0x0, 0x4, 0x1a, 0x8, 0x0, 0x0, 0x0, 0x8, 0x8, 0x8, 0x2, 0x0},
    },
}
&binlog.BinEvent{
    Header: &binlog.BinEventHeader{Timestamp:1536368222, EventType:0x2, ServerID:11111, EventSize:70, LogPos:177, Flag:0x8},
    Body:   &binlog.BinQueryEvent{
        SlaveProxyID:     69912308,
        ExecutionTime:    0,
        ErrorCode:        0x0,
        statusVarsLength: 26,
        StatusVars:       {0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x3, 0x73, 0x74, 0x64, 0x4, 0x21, 0x0, 0x21, 0x0, 0x2d, 0x0},
        Schema:           "test",
        Query:            "BEGIN",
    },
}
```

## To Be Continued...
