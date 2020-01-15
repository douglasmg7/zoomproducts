#!/usr/bin/env bash

if [ -z "$1" ]
  then
    echo "Usage: $0 product-id"
fi

curl -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/product/$1

# curl -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/product/123459789

printf "\n"