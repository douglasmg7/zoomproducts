#!/usr/bin/env bash

if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]; then
    echo "Usage: $0 ticket user password"
    exit 1
fi

curl -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" -X DELETE https://merchant.zoom.com.br/api/merchant/product/$1

printf "\n"