#!/usr/bin/env bash

# curl -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -d "5c83a2537b51490610f82be5" https://staging-merchant.zoom.com.br/api/merchant/product
# curl -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://staging-merchant.zoom.com.br/api/merchant/product

# curl -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products


RES=$(curl -s -u "zoomteste_zunka:H2VA79Ug4fjFsJb" -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

# echo $RES | jq -r .products
# echo $RES | jq -r '.products | .[0]'
echo $RES | jq -r '.products | .[] | select(.id == "123459789")'
	# jq '.[] | select(.id == "second")'
