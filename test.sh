# #!/bin/bash
curl --header "Content-Type: application/json" \
--request POST \
http://localhost:8080/api/v1/wallet
echo ""

curl --header "Content-Type: application/json" \
--request POST \
--data '{"time":"2022-09-18T07:25:40.20Z","from":"abcd","to":"bcda","amount":5}' \
http://localhost:8080/api/v1/wallet/abcd/send
echo ""

curl --header "Content-Type: application/json" \
--request GET \
http://localhost:8080/api/v1/wallet/abcd
echo ""

curl --header "Content-Type: application/json" \
--request POST \
--data '{"time":"2022-09-18T07:25:40.20Z","from":"abcd","to":"bcda","amount":5}' \
http://localhost:8080/api/v1/wallet/abcd/send
echo ""

curl --header "Content-Type: application/json" \
--request POST \
--data '{"time":"2022-09-18T07:25:40.20Z","from":"abcd","to":"bcda","amount":100}' \
http://localhost:8080/api/v1/wallet/abcd/send
echo "\ntest.sh: expected error, amount too big"

curl --header "Content-Type: application/json" \
--request GET \
http://localhost:8080/api/v1/wallet/abcd/history
echo ""
