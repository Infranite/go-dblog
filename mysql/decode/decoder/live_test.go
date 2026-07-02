package decoder

import (
	"testing"

	"github.com/Infranite/go-dblog/mysql/common"
)

func TestParseLiveDSN(t *testing.T) {
	cfg, err := parseLiveDSN("mysql://user:pass@[::1]:3310/app?server_id=42&binlog=mysql-bin.000001&pos=123&flavor=mariadb")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != "[::1]:3310" || cfg.user != "user" || cfg.password != "pass" || cfg.database != "app" {
		t.Fatalf("connection config = %#v", cfg)
	}
	if cfg.serverID != 42 || cfg.file != "mysql-bin.000001" || cfg.pos != 123 || cfg.flavor != "mariadb" {
		t.Fatalf("replication config = %#v", cfg)
	}

	cfg, err = parseLiveDSN("mysql://root@127.0.0.1/")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != "127.0.0.1:3306" || cfg.serverID != defaultLiveServerID || cfg.flavor != "mysql" {
		t.Fatalf("default config = %#v", cfg)
	}
}

func TestParseLiveDSNRejectsInvalidValues(t *testing.T) {
	for _, dsn := range []string{
		"",
		"postgres://root@127.0.0.1:3306/",
		"mysql://root@127.0.0.1:0/",
		"mysql://root@127.0.0.1:3306/?server_id=0",
		"mysql://root@127.0.0.1:3306/?pos=bad",
	} {
		if _, err := parseLiveDSN(dsn); err == nil {
			t.Fatalf("parseLiveDSN(%q) succeeded", dsn)
		}
	}
}

func TestDefaultEventTypeHeader(t *testing.T) {
	header := defaultEventTypeHeader()
	if header[common.TableMapEvent-1] != 8 {
		t.Fatalf("table map header length = %d", header[common.TableMapEvent-1])
	}
	if header[common.WriteRowsEventV2-1] != 10 || header[common.UpdateRowsEventV2-1] != 10 || header[common.DeleteRowsEventV2-1] != 10 {
		t.Fatalf("rows v2 header lengths = %d/%d/%d",
			header[common.WriteRowsEventV2-1],
			header[common.UpdateRowsEventV2-1],
			header[common.DeleteRowsEventV2-1],
		)
	}
}
