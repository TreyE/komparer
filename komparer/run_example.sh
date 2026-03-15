#!/bin/sh

ROOT_DIR=/Users/tevans/proj/mhc_k8s/
OLD_SHA=main
NEW_SHA=playground
ORIGINAL_DIR=`pwd`

cd $ROOT_DIR
git checkout $OLD_SHA
cd $ORIGINAL_DIR
go run cmd/build_dep_graph/main.go $ROOT_DIR ./old_data

cd $ROOT_DIR
git checkout $NEW_SHA
cd $ORIGINAL_DIR
go run cmd/build_dep_graph/main.go $ROOT_DIR ./current_data

go run cmd/evaluate_impact_graph/main.go old_data current_data
