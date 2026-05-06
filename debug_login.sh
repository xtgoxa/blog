#!/bin/bash
COOKIE=/tmp/cookies_blog.txt
rm -f $COOKIE

echo "=== GET login (full headers) ==="
curl -s -c $COOKIE -i http://localhost:8080/admin/login 2>&1 | head -15

echo ""
echo "=== Cookies saved ==="
cat $COOKIE 2>/dev/null || echo "no cookie file"
