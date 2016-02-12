#! /bin/sh

default_go_workspace=`pwd`

go_lib_deps='github.com/golang/protobuf/proto
github.com/golang/protobuf/protoc-gen-go
golang.org/x/net/context
google.golang.org/grpc'

bin_deps='protoc
go
'

# TODO
# 1. Add $GOBIN

saved_IFS=$IFS

# check the binary
IFS=$'\n'
for bin in $bin_deps
do
	if echo "$bin" | grep -E '^[[:blank:]]*$' > /dev/null 2>&1; then continue; fi

	if which $bin > /dev/null 2>&1; then
		continue
	fi

	echo "ERROR: please install \"$bin\" first" >&2 && exit 1
done
IFS=$saved_IFS

if [ -z "$GOPATH" ]; then
	echo "WARNING: \$GOPATH is not set, set to default path: ${default_go_workspace}" >&2
	export GOPATH=${default_go_workspace}
fi
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

if [ ! -e "$GOPATH" ]; then
	echo "WARNING: \$GOPATH does not exist, create the \"$GOPATH\"" >&2
	mkdir -p "$GOPATH" || (echo "ERROR: mkdir dir failed: $?" >&2 && exit 1)
else
	echo "INFO: \$GOPATH=$GOPATH is ok"
fi

IFS=$'\n'
for l in $go_lib_deps
do
	if echo "$l" | grep -E '^[[:blank:]]*$' > /dev/null 2>&1; then continue; fi

	go get "$l"
	if [ "$?" -ne 0 ]; then echo "ERROR: go get \"$l\" failed: $?" >&2; continue; fi
	go install "$l"
	if [ "$?" -ne 0 ]; then echo "ERROR: go install \"$l\" failed: $?" >&2; continue; fi
	echo "INFO: dep: \"$l\" is ok"
done
IFS=$saved_IFS

