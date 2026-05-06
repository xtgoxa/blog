#!/bin/bash
HOST=http://localhost:8080

echo "=== Test: non-admin user login (should fail) ==="
TOKEN=$(curl -s -c /tmp/cr.txt $HOST/admin/login | grep -oP 'value="[a-f0-9]{64}"' | head -1 | grep -oP '[a-f0-9]{64}')
echo "Token: $TOKEN"
R=$(curl -s -o /dev/null -w '%{http_code}' -b /tmp/cr.txt -c /tmp/cr.txt -X POST $HOST/admin/login -d "username=testuser&password=123456&csrf_token=$TOKEN")
echo "Result (should be 200, error page): $R"
curl -s -b /tmp/cr.txt -X POST $HOST/admin/login -d "username=testuser&password=123456&csrf_token=$TOKEN" | grep -i 'error\|invalid\|错误' | head -3
