# Blog Project v2.0

基于 Go + Iris + MariaDB 的博客系统

## 快速启动

### 方式一：在 Fedora WSL 中运行

```bash
# 1. 进入项目目录
cd /mnt/c/Users/xtgox/.qclaw/workspace/blog

# 2. 安装依赖
go mod tidy

# 3. 运行
go run .
```

### 方式二：在 Windows 中运行

```powershell
cd C:\Users\xtgox\.qclaw\workspace\blog
go run .
```

## 数据库配置

### Windows MariaDB 配置

1. 确保 MariaDB 服务正在运行
2. 创建数据库和表：
```bash
mysql -u root < init.sql
```

### WSL 连接 Windows MariaDB

WSL 通过 Windows 主机 IP 连接 MariaDB：
- 主机 IP: 在 WSL 中运行 `cat /etc/resolv.conf | grep nameserver`
- 默认配置在 `db.go` 中

如果需要修改数据库配置，编辑 `db.go` 中的 `DefaultConfig()` 函数。

## 访问地址

| 页面 | URL |
|------|-----|
| 首页 | http://localhost:8080 |
| 关于 | http://localhost:8080/about |
| 文章详情 | http://localhost:8080/post/1 |
| 管理后台 | http://localhost:8080/admin |
| 新建文章 | http://localhost:8080/admin/new |
| API接口 | http://localhost:8080/api/posts |

## 功能列表

- ✅ 首页文章列表
- ✅ 文章详情页
- ✅ 关于页面
- ✅ 管理后台（新建/编辑/删除文章）
- ✅ REST API 接口
- ✅ MariaDB 数据库支持
- ✅ 内存存储降级（数据库不可用时自动切换）

## 文件结构

```
blog/
├── main.go           # 主程序入口
├── db.go             # 数据库连接
├── repository.go     # 数据访问层
├── go.mod            # Go 模块定义
├── go.sum            # 依赖校验
├── init.sql          # 数据库初始化脚本
├── config.ini        # 配置文件模板
├── views/
│   ├── home/         # 前端页面
│   └── admin/        # 后台页面
└── public/css/       # 样式文件
```

## 新增功能 (v2.0)

1. **数据库支持** - MariaDB/MySQL 持久化存储
2. **自动降级** - 数据库不可用时自动切换内存存储
3. **删除功能** - 支持删除文章
4. **Repository 模式** - 数据访问层分离
