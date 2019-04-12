
all:	proto

proto:
	protoc --go_out=. kvstore/*.proto
	protoc --go_out=. chaosmonkey/*.proto
