# TermRelay 新服务器重建与灾难恢复手册

目标：即使原生产服务器完全不可用，也能用 GitHub 中的可信代码、固定的 TermRelay 镜像、独立保存的生产密钥和 PostgreSQL 备份重新建立服务。

这份流程用于合法的灾难恢复和服务器迁移，不用于绕过服务器所有者权限。

## 0. 恢复前必须拿到的东西

至少需要：

1. TermRelay GitHub 仓库访问能力；
2. 一份明确的正式镜像引用，例如：
   - `ghcr.io/yuchenm1303-png/sub2api:1.2.3`
   - 或更稳的 `ghcr.io/yuchenm1303-png/sub2api@sha256:...`
3. 当前生产 `.env` 的安全副本；
4. PostgreSQL 最近可用备份；
5. 如果备份是 `.age`：对应 `age` 私钥；
6. 最终切换流量所需的域名/DNS/Cloudflare 权限，或者一个新的可用域名。

密钥清单见 `RECOVERY_SECRETS_CN.md`。

## 1. 准备一台干净服务器

建议使用仍受安全更新支持的 Ubuntu LTS。

安装：

- Git
- Docker Engine
- Docker Compose plugin
- 如恢复 `.age` 备份：`age`

Docker 建议按照 Docker 官方仓库方式安装，不要使用来源不明的一键脚本。

确认：

```bash
docker version
docker compose version
git --version
```

## 2. 获取 TermRelay

```bash
git clone https://github.com/yuchenm1303-png/termrelay.git
cd termrelay/deploy
```

生产恢复时，最好 checkout 到经过确认的 tag/commit，而不是长期依赖一个不断变化的分支。

## 3. 恢复生产环境变量

把独立安全存储中的生产 `.env` 放到：

```text
termrelay/deploy/.env
```

权限建议：

```bash
chmod 600 .env
```

不要从公开仓库下载真实 `.env`，也不要把恢复后的 `.env` 提交回 Git。

## 4. 指定正式 TermRelay 镜像

不要使用 `latest`。

例如：

```bash
export TERMRELAY_IMAGE_REF='ghcr.io/yuchenm1303-png/sub2api:1.2.3'
```

更推荐恢复记录中保存 digest：

```bash
export TERMRELAY_IMAGE_REF='ghcr.io/yuchenm1303-png/sub2api@sha256:...'
```

## 5. 运行恢复前检查

如果数据库备份也已经放到本机：

```bash
export BACKUP_AGE_IDENTITY='/secure/path/termrelay-backup-key.txt'  # 仅加密备份需要
./recovery-preflight.sh /secure/path/termrelay-postgres-xxxx.dump.age
```

未准备数据库文件时也可以先检查配置：

```bash
./recovery-preflight.sh
```

检查会确认：

- Docker / Compose 可用；
- `.env`、基础 Compose、TermRelay overlay 存在；
- `POSTGRES_PASSWORD`、`JWT_SECRET`、`TOTP_ENCRYPTION_KEY` 不为空；
- 正式镜像来自 `ghcr.io/yuchenm1303-png/sub2api`；
- 不允许生产使用 `latest` / `main`；
- Compose 配置可以成功解析；
- 如果提供备份，则检查 SHA-256（如存在）。

## 6. 先启动基础数据库服务

```bash
docker compose \
  --env-file .env \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  up -d postgres redis
```

确认：

```bash
docker compose --env-file .env -f docker-compose.yml ps postgres redis
```

## 7. 恢复 PostgreSQL

恢复会重建当前 Compose PostgreSQL 中的目标数据库，因此必须明确确认：

```bash
export BACKUP_AGE_IDENTITY='/secure/path/termrelay-backup-key.txt'  # 仅 .age 需要
./restore-postgres.sh /secure/path/termrelay-postgres-xxxx.dump.age --confirm-restore
```

恢复脚本会：

1. 先验证备份；
2. 停止应用容器；
3. 重建目标 PostgreSQL 数据库；
4. 用 `pg_restore --exit-on-error` 导入；
5. 成功后重新启动应用；
6. 如果中途失败，保持应用停止，避免对外提供半恢复数据。

**不要在一个仍承载正常生产流量的数据库上随意运行这个命令。**

## 8. 按 TermRelay 正式 overlay 启动完整服务

```bash
docker compose \
  --env-file .env \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  up -d
```

查看：

```bash
docker compose \
  --env-file .env \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  ps
```

日志：

```bash
docker compose \
  --env-file .env \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  logs --tail=200 sub2api
```

## 9. 在切流量前验证

至少验证：

- `/health` 正常；
- 管理员可以登录；
- 普通用户数据数量合理；
- 关键上游账号/分组仍存在；
- 余额/订单/订阅抽样正确；
- API 请求可以正常完成；
- 2FA 用户不会因为 TOTP key 丢失而失效；
- 新服务器能执行一次新的 PostgreSQL 备份。

在这些验证完成之前，不要切换正式 DNS。

## 10. 切换域名/DNS

确认新环境完全健康后，再按项目授权流程切换 DNS / Cloudflare 到新服务器。

如果原域名无法控制，可先使用新的受控域名恢复业务连续性。域名切换是业务治理动作，不应由恢复脚本偷偷完成。

## 11. 恢复完成后立即做的事

1. 执行一次新的加密数据库备份；
2. 确认远端备份确实到达独立存储；
3. 记录当前镜像 digest；
4. 记录此次恢复使用的 commit/tag；
5. 检查 `.env` 权限；
6. 检查服务器没有保存 GitHub 管理员 Token；
7. 更新灾难恢复演练日期。

## 最终判断标准

真正完成灾备不是“有一个脚本”，而是满足：

```text
旧服务器完全消失
        ↓
新服务器 + GitHub + 密钥恢复包 + 数据库备份
        ↓
TermRelay 可以重新上线
```

如果这条链路能在测试环境完整走通，服务器就从“不可替代资产”变成“可替换运行环境”。
