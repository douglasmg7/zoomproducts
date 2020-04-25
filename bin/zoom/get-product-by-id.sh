#!/usr/bin/env bash

if [ -z "$1" ]
  then
    echo "Usage: $0 product-id"
    exit
fi

read -r USER PASS <<< $(./auth.sh)
RES=$(curl -s -u $USER:$PASS -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

# echo $RES | jq -r .products
# echo $RES | jq -r '.products | .[0]'
# echo $RES | jq -r '.products | .[] | select(.id == "123459789")'
echo $RES | jq -r '.products | .[] | select(.id == "'${1}'")'
printf "\n"