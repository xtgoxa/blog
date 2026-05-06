#!/bin/bash
COOKIE=/tmp/cookies_blog.txt
rm -f $COOKIE

# GET login page
echo "=== Step 1: GET login page ==="
curl -s -c $COOKIE http://localhost:8080/admin/login > /tmp/login_page.html
CSRF=$(grep -o 'name="csrf_token" value="[^"]*"' /tmp/login_page.html | sed 's/.*value="//;s/"$//')
echo "CSRF: $CSRF (len=${#CSRF})"

# POST login (NO -L follow, check response manually)
echo "=== Step 2: POST login ==="
curl -s -b $COOKIE -c $COOKIE -X POST http://localhost:8080/admin/login \
  --data-urlencode "username=admin" \
  --data-urlencode "password=admin123" \
  --data-urlencode "csrf_token=$CSRF" \
  -i 2>&1 | head -20

echo ""
echo "=== Step 3: GET users page ==="
curl -s -b $COOKIE -c $COOKIE http://localhost:8080/admin/users -o /tmp/users_page.html
echo "HTTP: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/admin/users)"
echo "Set-Cookie in response:"
curl -s -b $COOKIE -c $COOKIE -I http://localhost:8080/admin/users 2>&1 | grep -i "set-cookie\|blogsession"
