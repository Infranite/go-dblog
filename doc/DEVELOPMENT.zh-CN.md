# 开发

本文档记录 `go-dblog` 的本地检查、fixture 生成、pull request 和版本规则。

[English](./DEVELOPMENT.md)

## 要求

- Go 1.25 或更新版本。
- 本地 lint 需要 `golangci-lint`。
- Docker 只在本地调试 fixture 生成或集成测试时需要。

## 本地检查

运行本地单测：

```bash
make test
```

运行 lint：

```bash
make lint
```

运行 parser fuzz 和 benchmark smoke 门禁：

```bash
make fuzz-smoke
make bench-smoke
```

完整的 MySQL、MongoDB、PostgreSQL、Redis 真实 fixture 集成测试在 pull request CI
中运行。有 Docker 时才需要本地运行：

```bash
make integration
```

CI 还会把已测试 backend/version 矩阵和 parser benchmark 历史发布为 workflow
artifacts。当前矩阵和 artifact 名称见 [CI.zh-CN.md](./CI.zh-CN.md)。

## Fixture 调试

需要本地调试 CI 风格 fixture 生成时使用这些命令：

```bash
./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
./mysql/test/testdata/run_mysql_live.sh mysql:8.4
./mongo/testdata/generate_mongo_oplog.sh mongo:7.0
./mongo/testdata/run_mongo_live.sh mongo:7.0
./postgres/testdata/generate_postgres_logical.sh postgres:16
./postgres/testdata/run_postgres_live.sh postgres:16
./redis/testdata/generate_redis_aof.sh redis:7.2
./redis/testdata/run_redis_live.sh redis:7.2
```

## Pull Requests

贡献通过 pull request 处理。项目不单独维护 `CONTRIBUTING.md`。

- 提交前本地运行 `make test` 和受影响 module 的测试。
- parser 行为变更必须在对应 backend 中补测试。
- 用户可见行为变化需要同步更新相关 README 或 `doc/` 页面。
- 完整 fixture 集成测试、fuzz smoke、benchmark smoke、lint、vet 和漏洞扫描由 CI 运行。
- CI 会发布已测试矩阵和 benchmark history artifacts，作为 release evidence。
- Pull request 通过受保护的 `ci` 和 `merge-policy` 检查后合入。

## 版本

- 根模块 tag 使用 `vX.Y.Z`。
- backend module tag 使用 `mysql/vX.Y.Z`、`postgres/vX.Y.Z`、
  `mongo/vX.Y.Z`、`redis/vX.Y.Z`。
- backend module 在每个 release tag 中跟随根模块版本。
- GitHub Releases 和 git tags 是公开发布记录。
- Git history 是详细变更记录；项目不单独维护 release notes 或 changelog 文件。
