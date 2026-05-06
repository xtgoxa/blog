#!/bin/bash
HOST=http://localhost:8080

echo "=== Test 1: admin with username (detail) ==="
TOKEN1=$(curl -s -c /tmp/c1.txt $HOST/admin/login | grep -oP 'value="[a-f0-9]{64}"' | head -1 | grep -oP '[a-f0-9]{64}')
echo "Token: $TOKEN1"
curl -s -b /tmp/c1.txt -c /tmp/c1.txt -X POST $HOST/admin/login -d "username=admin&password=admin123&csrf_token=$TOKEN1" | grep -i 'error\|invalid\|错误' | head -5
echo "HTTP: $?"

echo ""
echo "=== Check admin user in DB ==="
/mnt/d/MariaDB/bin/mysql.exe -u root -p331563615 blog -e "SELECT id,username,role,LEFT(password,20) FROM users WHERE username='admin' OR role='admin';"
