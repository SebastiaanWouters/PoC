
#init
go get -d -v ./...

#build
EGOPATH=/snap/ego-dev/current/opt/ego CGO_CFLAGS=-I$EGOPATH/include CGO_LDFLAGS=-L$EGOPATH/lib go build

#run
./client -port 6666