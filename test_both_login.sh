#!/bin/bash
HOST=localhost:8080
# Test 1: admin with username field (new way)
TOKEN1=$(curl -s -c /tmp/c1.txt $HOST/admin/login | grep -oP 'value="[a-f0-9]{64}"' | head -1 | grep -oP '[a-f0-9]{64}')
echo "Token1: $TOKEN1"
R1=$(curl -s -o /dev/null -w '%{http_code}' -b /tmp/c1.txt -c /tmp/c1.txt -X POST $HOST/admin/login -d "username=admin&password=admin123&csrf_token=$TOKEN1")
echo "Test1 (admin+username): $R1"

# Test 2: admin without username (old way)
TOKEN2=$(curl -s -c /tmp/c2.txt $HOST/admin/login | grep -oP 'value="[a-f0-9]{64}"' | head -1 | grep -oP '[a-f0-9]{64}')
echo "Token2: $TOKEN2"
R2=$(curl -s -o /dev/null -w '%{http_code}' -b /tmp/c2.txt -c /tmp/c2.txt -X POST $HOST/admin/login -d "password=admin123&csrf_token=$TOKEN2")
echo "Test2 (admin only): $R2"

# Test 3: wrong password
TOKEN3=$(curl -s -c /tmp/c3.txt $HOST/admin/login | grep -oP 'value="[a-f0-9]{64}"' | head -1 | grep -oP '[a-f0-9]{64}')
R3=$(curl -s -o /dev/null -w '%{http_code}' -b /tmp/c3.txt -c /tmp/c3.txt -X POST $HOST/admin/login -d "username=admin&password=wrong&csrf_token=$TOKEN3")
echo "Test3 (wrong pass): $R3"
