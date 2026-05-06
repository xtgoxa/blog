#!/bin/bash
mysql -h 172.31.224.1 -u zgoxa -pzgoxa blog --default-auth=mysql_native_password -e "SELECT id, username, role, is_admin FROM users WHERE username='admin';"
echo "---ALL USERS---"
mysql -h 172.31.224.1 -u zgoxa -pzgoxa blog --default-auth=mysql_native_password -e "SELECT id, username, role, is_admin FROM users;"
