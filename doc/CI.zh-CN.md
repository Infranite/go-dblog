# CI 证据

`go-dblog` 将 pull request CI 作为 backend 行为的发布门禁。本地检查刻意保持较短；
依赖真实 fixture 和服务的覆盖在 GitHub Actions 中运行。

[English](./CI.md)

## 必需检查

受保护的 `ci` job 要求以下检查通过后 pull request 才能合入：

| 范围 | 覆盖内容 |
|---|---|
| 单测和 race tests | 根 package，以及 MySQL、PostgreSQL、MongoDB、Redis module 的 short tests。 |
| fixture-backed integration | MySQL 5.6、5.7、8.0、8.4 binlog fixture；PostgreSQL 16 logical decoding；MongoDB 7.0 oplog/change streams；Redis 7.2 AOF/PSYNC。 |
| Parser robustness | MySQL、PostgreSQL、MongoDB、Redis parser 的 fuzz smoke targets。 |
| Parser performance | MySQL、PostgreSQL、MongoDB、Redis parser 的 benchmark smoke targets。 |
| 静态门禁 | 每个 module 都运行 `golangci-lint`、`go vet` 和 `govulncheck`。 |
| CI evidence | `ci-report` 生成已测试矩阵和 benchmark history artifacts。 |

## 发布产物

每次 CI run 都会发布保留 90 天的小型文本 artifact：

| Artifact | 文件 | 用途 |
|---|---|---|
| `benchmark-<module>` | `benchmark-<module>.txt` | 单个 parser module 的原始 Go benchmark 输出。 |
| `ci-report` | `tested-matrix.md`、`tested-matrix.json`、`benchmarks.md`、`benchmarks.jsonl`、`ci-report.md` | 面向人工和机器读取的 release evidence。 |

workflow summary 也会包含同一份 `ci-report.md` 内容，因此 reviewer 不需要打开 job logs
就能查看已测试 backend/version 矩阵和 parser benchmark 历史。

## 当前矩阵

| 产品 | Runtime | CI 范围 |
|---|---|---|
| 公共 API | Go 1.25.x | 根 package 测试、registry 测试、checkpoint 测试。 |
| MySQL | MySQL 5.6、5.7、8.0、8.4 | binlog fixture 生成和 module tests；MySQL 8.4 额外覆盖 live replication 和 decoder benchmark。 |
| PostgreSQL | PostgreSQL 16 | logical decoding fixture 生成、live SQL polling、wire replication reader 和 module tests。 |
| MongoDB | MongoDB 7.0 | oplog fixture 生成、live change streams 和 module tests。 |
| Redis | Redis 7.2 | AOF fixture 生成、live PSYNC replication stream 和 module tests。 |

## 保留范围

CI artifacts 刻意保持小型文本格式。它们在 GitHub Actions 免费 artifact 保留窗口内提供
近期 benchmark history，不保存生成的数据库 fixture 或服务日志。
