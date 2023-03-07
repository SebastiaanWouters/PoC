
#init
go get node

#build
EGOPATH=/snap/ego-dev/current/opt/ego CGO_CFLAGS=-I$EGOPATH/include CGO_LDFLAGS=-L$EGOPATH/lib go build

#run
./node -port 6666