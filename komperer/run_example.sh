#!/bin/sh

ROOT_DIR=/Users/tevans/proj/cme_k8s/
OLD_SHA=8946a9e9a57be870dd8bd77bf03c46f8a79c2224
ORIGINAL_DIR=`pwd`

cd $ROOT_DIR
git checkout $OLD_SHA
cd $ORIGINAL_DIR
go run cmd/build_dep_graph/main.go $ROOT_DIR ./old_data

cd $ROOT_DIR
git checkout trunk
git diff --no-renames --name-status $OLD_SHA > $ORIGINAL_DIR/example_diffs.tsv

cd $ORIGINAL_DIR
go run cmd/build_dep_graph/main.go $ROOT_DIR ./current_data

go run cmd/evaluate_impact_graph/main.go current_data old_data example_diffs.tsv
