#!/usr/bin/env bash

read -r USER PASS <<< $(./auth.sh)
RES=$(curl -s -u $USER:$PASS -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)
echo $RES | jq -r '.products | .[]'
