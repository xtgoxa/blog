#!/bin/bash
# 找一个已有 session 的用户（普通用户），测试注册新 admin 账号
# 先查看注册路由
echo "=== Test if register works ==="
curl -s -c /tmp/c.txt http://localhost:8080/register -o /dev/null
echo "Register page status: $?"

echo ""
echo "=== List all users via API ==="
curl -s http://localhost:8080/api/posts | head -100
