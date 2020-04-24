#!/usr/bin/env bash

# echo '.products | .[] | select(.id == "'$1'")'
# exit

if [ -z "$1" ]
  then
    echo "Usage: $0 product-id"
    exit
fi

RES=$(curl -s -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

# echo $RES | jq -r .products
# echo $RES | jq -r '.products | .[0]'
# echo $RES | jq -r '.products | .[] | select(.id == "123459789")'
echo $RES | jq -r '.products | .[] | select(.id == "'${1}'")'

printf "\n"