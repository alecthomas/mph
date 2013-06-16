chd.pb.go: chd.proto
	protoc --go_out=. chd.proto
