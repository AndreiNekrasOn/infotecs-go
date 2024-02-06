# #!/bin/bash
curl --header "Content-Type: application/json" \
--request POST \
http://localhost:8080/api/v1/wallet

curl --header "Content-Type: application/json" \
--request POST \
--data '{"from":"abcd","to":"bcda","amount":5}' \
http://localhost:8080/api/v1/wallet/abcd/send

curl --header "Content-Type: application/json" \
--request GET \
http://localhost:8080/api/v1/wallet/abcd

curl --header "Content-Type: application/json" \
--request POST \
--data '{"from":"abcd","to":"bcda","amount":5}' \
http://localhost:8080/api/v1/wallet/abcd/send

curl --header "Content-Type: application/json" \
--request POST \
--data '{"from":"abcd","to":"bcda","amount":100}' \
http://localhost:8080/api/v1/wallet/abcd/send

curl --header "Content-Type: application/json" \
--request GET \
http://localhost:8080/api/v1/wallet/abcd/history
