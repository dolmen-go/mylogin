#!/bin/bash

for f in 0 1 2 3 4 5 6 7 8 9
do
    [[ -f "$f.cnf" ]] && continue
    MYSQL_TEST_LOGIN_FILE="$f.cnf" mysql_config_editor set
done

for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16
do
    f=$(printf 'padding%02d.cnf' $i )
    [[ -f "$f" ]] && continue
    MYSQL_TEST_LOGIN_FILE="$f" mysql_config_editor set --login-path=$(expr substr 0123456789abcdef012 1 $(( 17 - ( $i + 3 ) % 16 )) )
done
