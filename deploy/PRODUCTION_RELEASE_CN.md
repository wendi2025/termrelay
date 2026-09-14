# TermRelay 正式版本与生产部署规则

这份规则的目的很简单：**生产服务器只负责运行，正式软件版本由 TermRelay 自己的 GitHub Release / GHCR 提供。**

这样即使服务器以后更换，TermRelay 的正式版本来源仍然不会跟着服务器一起丢失。

## 1. 当前为什么需要覆盖文件

仓库原有 `docker-compose.yml` 为了兼容上游，默认使用：

```text
weishaw/sub2api:latest
```

`latest` 是可变标签，而且镜像来源属于上游项目，不适合直接作为 TermRelay 的长期生产版本来源。

因此新增：

```text
docker-compose.termrelay.yml
```

它不会修改原 Compose 文件，而是在正式部署时覆盖应用镜像。

## 2. 正式部署只使用本仓库发布的镜像

本仓库的 Release 工作流会把镜像发布到：

```text
ghcr.io/yuchenm1303-png/sub2api
```

部署某一个明确版本时：

```bash
cd deploy
export TERMRELAY_IMAGE_REF=ghcr.io/yuchenm1303-png/sub2api:1.2.3

docker compose \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  pull sub2api

docker compose \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  up -d
```

> 上面的 `1.2.3` 只是示例。应使用 TermRelay GitHub Release 实际发布出来的镜像标签。

不要在正式环境里把 `latest` 当作唯一版本记录。

## 3. 更稳的方式：固定镜像 digest

版本标签通常已经足够好，但生产环境最好进一步记录镜像 digest。

先拉取明确版本：

```bash
export TERMRELAY_IMAGE_REF=ghcr.io/yuchenm1303-png/sub2api:1.2.3
docker pull "$TERMRELAY_IMAGE_REF"
```

查看不可变的 RepoDigest：

```bash
docker image inspect "$TERMRELAY_IMAGE_REF" --format '{{json .RepoDigests}}'
```

然后可以把正式部署变量改为类似：

```bash
export TERMRELAY_IMAGE_REF='ghcr.io/yuchenm1303-png/sub2api@sha256:...'
```

这样以后即使某个同名标签发生变化，服务器仍会运行你当时审核过的那一个镜像内容。

## 4. 回滚

升级前记录当前的 `TERMRELAY_IMAGE_REF`。

如果新版本有问题，只需要把变量换回上一版，再执行：

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  pull sub2api

docker compose \
  -f docker-compose.yml \
  -f docker-compose.termrelay.yml \
  up -d
```

数据库发生 schema migration 的版本，回滚前还必须先确认数据库迁移是否支持回退，不能只回滚容器镜像。

## 5. 服务器不需要 GitHub 管理权限

生产服务器不应该保存 GitHub 主账号密码、仓库管理 Token 或能够删除/修改仓库的凭据。

如果 GHCR 镜像是公开的，服务器直接拉取即可。

如果以后镜像改成私有，服务器也只应使用最小权限的只读 package 凭据，而不是仓库管理员凭据。

## 6. 正式发布建议流程

推荐固定成：

```text
代码进入 main
    ↓
CI / Security Scan 通过
    ↓
创建正式 tag / GitHub Release
    ↓
GitHub Actions 构建 GHCR 镜像
    ↓
记录明确版本或 digest
    ↓
生产服务器部署该镜像
```

也就是说：

```text
GitHub / Release = 软件版本源头
服务器            = 可替换的运行机器
```

## 7. 不要把这个机制做成秘密开关

生产镜像控制的作用是保证版本可信、可回滚、可重建，不是远程关闭合作方服务器。

不要加入：

- 隐藏停服逻辑；
- 远程删库；
- 到期自毁；
- 只有某个人知道的绕权 Token；
- 从 GitHub 对线上用户实施秘密控制的代码。

真正的主动权来自你始终拥有可信代码、正式发布历史、可恢复的数据和重新部署能力。
