
all:	proto

proto:
	protoc -I chaosmonkey -I${GOPATH}/src --go_out=plugins=grpc:chaosmonkey chaosmonkey/chaosmonkey.proto
	protoc -I kvstore -I${GOPATH}/src --go_out=plugins=grpc:kvstore kvstore/kvstore.proto
	protoc -I raft -I${GOPATH}/src --go_out=plugins=grpc:raft raft/raft.proto
