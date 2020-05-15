#!/usr/bin/env bash
    
read -r USER PASS <<< $(./auth.sh)
# echo curl -s -u $USER:$PASS -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products
RES=$(curl -s -u $USER:$PASS -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

STATUS=$(echo $RES | jq -r '.status')

if [ $STATUS != null ]; then
    echo $RES
    exit 0
fi

echo $RES | jq -r '.products | .[] | select(.active == true) | .id'

# echo $RES | jq -r .products
# echo $RES | jq -r '.products | .[0]'
# echo $RES | jq -r '.products | .[] | select(.id == "123459789")'


# echo $RES | jq -r '.products | .[]'
	# jq '.[] | select(.id == "second")'

