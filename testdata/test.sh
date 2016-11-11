#!/bin/bash

for f in 0 1 2 3 4 5 6 7 8 9
do
    MYSQL_TEST_LOGIN_FILE="$f.cnf" mysql_config_editor set
done
