# TermRelay 数据备份与恢复底线

这份文档的目标不是让某一方拥有“隐藏后门”，而是保证生产服务器、域名或合作关系发生故障时，TermRelay 的业务数据不会只剩唯一一份。

## 1. 控制权原则

生产服务器应被视为“可替换机器”，而不是 TermRelay 唯一的数据归宿。

至少应满足：

- GitHub 源码和发布历史不依赖生产服务器保存。
- PostgreSQL 每天至少生成一次可验证备份。
- 至少一份备份自动复制到生产服务器之外、由项目方独立控制的存储。
- 远端备份默认加密；生产服务器只保存加密公钥，不保存解密私钥。
- `.env`、`.backup.env`、数据库 dump、私钥都禁止提交到 Git。
- 定期在另一台机器验证备份能够被读取，避免“有备份但不能恢复”。

## 2. 备份脚本

仓库提供：

- `backup-postgres.sh`：生成 PostgreSQL custom-format dump，立即做结构校验，可选 age 加密及 rclone 异地上传。
- `verify-postgres-backup.sh`：校验 SHA-256，并验证 PostgreSQL 归档结构。
- `.backup.env.example`：备份配置模板；复制出的 `.backup.env` 已被 Git 忽略。

脚本不会读取或上传 GitHub 管理令牌，也不会修改用户、余额、订单或业务配置。

## 3. 首次配置

在生产服务器的 `deploy` 目录执行：

```bash
cp .backup.env.example .backup.env
chmod 600 .backup.env
```

编辑 `.backup.env`。

建议至少设置：

```bash
BACKUP_RETENTION_DAYS=7
BACKUP_AGE_RECIPIENT=age1...你的加密公钥...
BACKUP_RCLONE_REMOTE=你的远端名称:termrelay/postgres
ALLOW_UNENCRYPTED_REMOTE_BACKUP=false
```

### 为什么只把 age 公钥放到服务器

备份服务器只需要“加密”能力，不需要“解密”能力。

因此推荐：

1. 在你自己控制的电脑或恢复机生成 age 密钥。
2. 把 **public recipient** 写入生产服务器的 `.backup.env`。
3. 私有 identity 文件只保存在你自己的安全位置，并额外做离线副本。

这样即使生产服务器以后由别人完全控制，从远端拿到的备份仍然不能仅凭服务器上的信息解密。

> 注意：数据库包含用户、账号、订单、API 配置等敏感业务数据。解密密钥必须严格保护，并遵守适用的隐私、数据保护及合作约定。

## 4. 手动执行一次备份

```bash
cd deploy
bash backup-postgres.sh
```

成功时会生成备份及 `.sha256` 校验文件。

如果配置了 `BACKUP_RCLONE_REMOTE`，脚本会把最终产物复制到远端。未配置 age 加密时，脚本默认拒绝远端上传。

## 5. 验证备份

未加密备份：

```bash
bash verify-postgres-backup.sh backups/termrelay-postgres-YYYYMMDDTHHMMSSZ.dump
```

加密备份建议在你自己的恢复机器执行，并临时指定私钥：

```bash
export BACKUP_AGE_IDENTITY=/secure/path/termrelay-backup-key.txt
bash verify-postgres-backup.sh /path/to/termrelay-postgres-YYYYMMDDTHHMMSSZ.dump.age
```

验证成功只说明 dump 完整、PostgreSQL 能识别归档结构。正式上线前还应定期进行一次“恢复演练”：恢复到独立测试数据库并检查关键表和记录数量。

## 6. 自动执行

确认手动备份成功后，可以使用 cron 或 systemd timer 每天调用：

```bash
cd /你的/termrelay/deploy && bash backup-postgres.sh
```

不要在定时任务命令里直接写数据库密码、GitHub Token 或 age 私钥。

## 7. 建议保留策略

初期可以采用：

- 生产服务器本地：最近 7 天。
- 异地存储：至少保留更长时间，并开启版本历史/防误删能力（如果存储服务支持）。
- 每月至少抽查一次备份完整性。
- 每次重大数据库迁移、支付系统调整、批量数据操作之前额外做一次备份。

## 8. 灾难恢复时的顺序

如果原服务器不可用：

1. 从你控制的 GitHub 获取可信版本代码。
2. 准备新的 Linux/Docker 环境。
3. 使用仓库中的 Compose 配置启动 PostgreSQL、Redis 和 TermRelay。
4. 从你控制的远端存储取回数据库备份。
5. 在恢复机使用 age 私钥解密。
6. 先恢复到测试数据库并验证，再切换正式服务。
7. 最后处理 DNS/域名切换。

域名和服务器可能更换，但源码、发布历史、备份和恢复能力不能只有一方持有。

## 9. 明确禁止

不要用灾难恢复机制实现：

- 隐藏停服开关；
- 远程删库；
- 秘密管理员后门；
- 未经授权导出或读取用户数据；
- 能绕过正常权限体系的万能密钥。

真正可靠的主动权来自“可独立恢复”，而不是“能破坏现网”。
