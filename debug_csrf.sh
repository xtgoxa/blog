#!/bin/bash
COOKIE=/tmp/cookie_jar.txt
rm -f $COOKIE

echo "=== 1. GET login page ==="
curl -s -c $COOKIE http://localhost:8080/admin/login > /tmp/login_page.html
echo "Status: $?"
CSRF=$(grep -o 'name="csrf_token" value="[^"]*"' /tmp/login_page.html | sed 's/.*value="//;s/"$//')
echo "CSRF from HTML: $CSRF"
echo "Cookie file:"
cat $COOKIE

echo ""
echo "=== 2. POST login ==="
RESPONSE=$(curl -s -D - -b $COOKIE -c $COOKIE http://localhost:8080/admin/login \
  --data-urlencode "username=admin" \
  --data-urlencode "password=admin123" \
  --data-urlencode "csrf_token=$CSRF" 2>&1)
echo "$RESPONSE" | grep -E "^HTTP|^Location|Forbidden"

echo ""
echo "=== 3. GET users (verify login) ==="
curl -s -b $COOKIE -c $COOKIE -w "HTTP_STATUS:%{http_code}\n" -o /tmp/users.html http://localhost:8080/admin/users
