#!/usr/bin/env bash
if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 user password"
    exit 1
fi

RES=$(curl -s -u "$1:$2" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

STATUS=$(echo $RES | jq -r '.status')

if [ $STATUS != null ]; then
    echo $RES
    exit 0
fi

echo $RES | jq -r '.products | .[] | select(.active == true)'

# RES=$(curl -s -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

# echo $RES | jq -r .products
# echo $RES | jq -r '.products | .[0]'
# echo $RES | jq -r '.products | .[] | select(.id == "123459789")'


# echo $RES | jq -r '.products | .[]'
	# jq '.[] | select(.id == "second")'

