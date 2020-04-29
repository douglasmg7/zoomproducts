#!/usr/bin/env bash

if [ -z "$1" ]; then
    echo "Usage: $0 product-id"
    exit 1
fi

read -r USER PASS <<< $(./auth.sh)
curl -u $USER:$PASS -H "Content-Type: application/json" -X DELETE https://merchant.zoom.com.br/api/merchant/product/$1

printf "\n"