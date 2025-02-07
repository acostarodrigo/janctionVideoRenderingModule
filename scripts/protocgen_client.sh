#!/usr/bin/env bash

set -e
echo "Generating gogo proto code"
cd proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    echo $file

    # Skip google/protobuf files to avoid conflicts
    if [[ $file == *"google/protobuf/"* ]]; then
      echo "Skipping $file to avoid conflicts with protobufjs"
      continue
    fi

    # this regex checks if a proto file has its go_package set to github.com/janction/videoRendering/api/...
    # gogo proto files SHOULD ONLY be generated if this is false
    # we don't want gogo proto to run for proto files which are natively built for google.golang.org/protobuf
    if grep -q "option go_package" "$file" && grep -H -o -c 'option go_package.*github.com/janction/videoRendering/api' "$file" | grep -q ':0$'; then
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

echo "Generating Client proto code"
buf generate --template buf.gen.client.yaml --path ./janction/videoRendering/v1/
