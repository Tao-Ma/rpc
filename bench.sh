#!/bin/sh

bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 1 -req_num 147035
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 2 -req_num 259814
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 3 -req_num 386114
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 4 -req_num 410275
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 6 -req_num 472222
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 8 -req_num 480160

bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 100 -req_num 2000000
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 1 -burst_num 500 -req_num 3000000
bin/client -address '127.0.0.1:10000' -router_per_conn -conn_num 8 -burst_num 500 -req_num 3000000
