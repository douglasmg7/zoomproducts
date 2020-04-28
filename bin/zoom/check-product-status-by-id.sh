#!/usr/bin/env bash
if [ -z "$1" ]
  then
    echo "Usage: $0 product-id"
    exit
fi

read -r USER PASS <<< $(./auth.sh)

while [ : ]
do
    NOW=`date`
    RES=$(curl -s -u $USER:$PASS -H "Content-Type: application/json" https://merchant.zoom.com.br/api/merchant/products)

    # echo $RES | jq -r .products
    # echo $RES | jq -r '.products | .[0]'
    # echo $RES | jq -r '.products | .[] | select(.id == "123459789")'

    # RES=`$RES | jq -r '.products | .[] | select(.id == "'${1}'") | .active'`
    # printf "$NOW\n"
    # echo $RES | jq -r '.products | .[] | select(.id == "'${1}'") | .active'
    OUT=$(echo $RES | jq -r '.products | .[] | select(.id == "'${1}'") | .active')
    printf "$OUT\t$NOW\n"
    # printf "\n"

    # printf "$NOW\t$RES\n"
    sleep 20m
done





