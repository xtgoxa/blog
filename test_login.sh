#!/bin/bash
BASE="http://localhost:8080"
CS=/tmp/cs$$cookie

# GET login page to get CSRF token
curl -s -c $CS "$BASE/admin/login" -o /tmp/login_page.html
TOKEN=$(grep -oP 'name="csrf_token"[^>]*value="\K[^"]+' /tmp/login_page.html)
echo "Token: $TOKEN"

# POST login
curl -s -b $CS -c $CS \
  -X POST "$BASE/admin/login" \
  -d "csrf_token=$TOKEN" \
  -d "username=admin" \
  -d "password=admin123" \
  -o /tmp/login_resp.html \
  -w "HTTP:%{http_code}\n"

# Check admin index
curl -s -b $CS -o /dev/null -w "Admin index: %{http_code}\n" "$BASE/admin"

rm -f $CS /tmp/login_page.html /tmp/login_resp.html