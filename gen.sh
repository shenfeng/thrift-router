#! /bin/bash

cp ~/workspace/engine/data.thrift .

python gen_hooks.py --file data.thrift --mode gen-go > hooks.go
python gen_hooks.py --file data.thrift --mode gen-java > clients.java

rm -rf gen-go

thrift -gen go:thrift_import=thrift data.thrift

rm -rf ~/gocode/src/data
mv gen-go/data ~/gocode/src/
rm -rf gen-go

rm gen-py -rf
thrift -gen py data.thrift
