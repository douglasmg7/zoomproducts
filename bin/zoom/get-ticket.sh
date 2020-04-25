#!/usr/bin/env bash

if [ -z "$1" ]
  then
    echo "Usage: $0 ticket"
fi

read -r USER PASS <<< $(./auth.sh)
curl -u $USER:$PASS -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/receipt/$1

printf "\n"