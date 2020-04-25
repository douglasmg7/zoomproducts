#!/usr/bin/env bash
    
# Development mode.
TOKEN_USER=zoomUserDevelopment
TOKEN_PASS=zoomPassDevelopment

# Production mode.
if [[ $RUN_MODE == production ]]; then
    TOKEN_USER=zoomUserProduction
    TOKEN_PASS=zoomPassProduction
fi

USER=`grep "^\s*$TOKEN_USER" ../../s.go | cut -d " " -f3 | sed 's/"//g'`
PASS=`grep "^\s*$TOKEN_PASS" ../../s.go | cut -d " " -f3 | sed 's/"//g'`

echo $USER $PASS
