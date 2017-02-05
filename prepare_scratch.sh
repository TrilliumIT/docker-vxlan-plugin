#!/bin/bash

###
# Helper script to retrieve
# the bare minimum files
# in order to build a small container
#
# Meant to be run in the build container
###
# Two possibities tried here:
# - one with the buse go binary,
# that still requires additionnal libs
# - one with a specially-built static binary
###


# Shared directory to put all files in
# = a docker volume
export_dir="/export"


copy_dynamic() {
### 
# Get the base go binary,
# and its system lib dependencies
###
  # List of required files
  files_list=(
  /go/bin/docker-vxlan-plugin
  /lib/x86_64-linux-gnu/ld-2.13.so
  /lib/x86_64-linux-gnu/libc-2.13.so
  /lib/x86_64-linux-gnu/libdl-2.13.so
  /lib/x86_64-linux-gnu/libpthread-2.13.so
  )

  # Copy files to export dir
  for file in ${files_list[@]}; do
    echo $file
    mkdir -p "$export_dir"/"$(dirname "$file")"
    cp "$file" "$export_dir"/"$file"
  done
}


build_copy_static() {
###
# Build a static binary,
# so that it requires no additional library
###
  # Build static binary
  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo
  #copy it to export dir
  cp docker-vxlan-plugin "$export_dir"/
}


### Either one or the other function
#copy_dynamic
build_copy_static

