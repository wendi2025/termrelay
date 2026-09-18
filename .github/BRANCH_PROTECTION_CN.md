# TermRelay `main` 分支保护建议

目标不是把仓库锁死，而是防止任何合作方、误操作或泄露的凭据直接改写项目历史。

当前 TermRelay 的 `main` 分支没有保护。建议在 GitHub 仓库设置里逐步开启以下规则。

## 第一阶段：现在就可以开启

优先开启这些不会把仓库所有者自己锁在门外的规则：

1. **禁止 force push（强制推送）**
2. **禁止删除 `main` 分支**
3. **合并前要求状态检查通过**
   - CI
   - Security Scan
   - Production Control

这些规则的作用是：即使某个协作者拥有写权限，也不能悄悄重写主分支历史；正式变更至少要经过自动测试。

## 第二阶段：确认协作流程后再开启

可以考虑要求所有 `main` 变更通过 Pull Request。

但是在只有一个主要维护者时，不建议立刻强制“必须由另一个 CODEOWNER 批准”，否则仓库所有者可能无法批准自己的 PR，反而把自己锁住。

如果未来有第二个长期可信维护者，再考虑：

- Require pull request reviews
- Require review from Code Owners
- Dismiss stale approvals when new commits are pushed

## Release 工作流兼容性

开启“所有 `main` 变更必须经过 Pull Request”之前，必须同时检查 Release 工作流。

当前上游式 Release 流程在发布成功后会尝试把 `backend/cmd/server/VERSION` 直接 push 回默认分支。严格分支保护启用后，这种直接 push 应当被禁止，而不是给发布机器人仓库管理员级绕过权限。

TermRelay 的目标做法是：正式 tag / Release / GHCR 镜像负责记录已发布版本；如需同步 `VERSION` 文件，应通过普通 Pull Request 完成。生产服务器不需要、Release 机器人也不应获得绕过仓库治理所需的长期高权限凭据。

在这一兼容性调整合并前，不要为了让 Release 继续工作而给 GitHub Actions 配置可任意绕过 `main` 保护的管理权限。

## 不建议

不要为了控制权：

- 把 GitHub 主账号密码交给服务器或合作方；
- 在服务器保存拥有仓库管理权限的 Personal Access Token；
- 允许自动部署凭据删除仓库、修改成员或更改仓库设置；
- 用隐藏脚本绕过 GitHub 正常权限模型。

生产服务器如果只需要公开代码/公开 GHCR 镜像，就不需要 GitHub 管理权限。

## CODEOWNERS 的作用

仓库中的 `.github/CODEOWNERS` 会明确关键路径的默认负责人是 `@yuchenm1303-png`。

它本身只是“归属声明/自动请求审核”，只有配合 GitHub 分支保护规则时才会变成强制审核规则。因此可以先提交 CODEOWNERS，再逐步打开分支保护。
