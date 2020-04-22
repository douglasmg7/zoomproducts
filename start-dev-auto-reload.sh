#!/usr/bin/env bash
[[ `systemctl status mongodb | awk '/Active/{print $2}'` == inactive ]] && sudo systemctl start mongodb
CompileDaemon -build="go build" -recursive="true" -command="./zoomproducts dev"