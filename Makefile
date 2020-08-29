.PHONY: all clean base check coverage install

PREFIX=/usr
SBINDIR=$(DESTDIR)$(PREFIX)/sbin
PKGPATH=pkg
CURDIR=$(shell pwd)
XARCH=$(shell arch)
X86_64=x86_64
ARRCH64=aarch64
VERSION=1.0

all: wisdomd wisdom

wisdomd:
	export GOPATH=`cd ../../../;pwd`;\
	export PATH=$$GOPATH/bin:$$PATH;\
	rm -rf tmp1;\
	mkdir tmp1;\
	go build -mod=vendor -ldflags '-w -s -extldflags=-static -extldflags=-zrelro -extldflags=-znow -buildmode=pie -tmpdir tmp1 -X main.version=${VERSION}' -v -o $$GOPATH/pkg/wisdomd $$GOPATH/src/gitee.com/wisdom-advisor/cmd/wisdomd

wisdom:
	export GOPATH=`cd ../../../;pwd`;\
	export PATH=$$GOPATH/bin:$$PATH;\
	rm -rf tmp2;\
	mkdir tmp2;\
	go build -mod=vendor -ldflags '-w -s -extldflags=-static -extldflags=-zrelro -extldflags=-znow -buildmode=pie -tmpdir tmp2 -X main.version=${VERSION}' -v -o $$GOPATH/pkg/wisdom $$GOPATH/src/gitee.com/wisdom-advisor/cmd/wisdom

format :
	export GOPATH=`cd ../../..;pwd` && gofmt -e -s -w cmd
	export GOPATH=`cd ../../..;pwd` && gofmt -e -s -w common

clean:
	export GOPATH=`cd ../../..;pwd`;\
	rm -rf $$GOPATH/$(PKGPATH)/*;\
	rm -rf $$GOPATH/bin/*;\
	rm -rf ./cov

check:
	export GOPATH=`cd ../../..;pwd`;\
	export PATH=$$GOPATH/bin:$$PATH;\
	go test -mod=vendor ./... -count=1

coverage:
	export GOPATH=`cd ../../..;pwd`;\
	export PATH=$$GOPATH/bin:$$PATH;\
	export GOFLAGS="$$GOFLAGS -mod=vendor";\
	rm -rf ./cov;\
	mkdir cov;\
	go test -coverpkg=./... -coverprofile=./cov/coverage.data ./cmd/wisdomd \
		./common/cpumask ./common/policy ./common/procscan ./common/sched \
		./common/sysload ./common/topology ./common/utils;\
	go tool cover -html=./cov/coverage.data -o ./cov/coverage.html;\
	go tool cover -func=./cov/coverage.data -o ./cov/coverage.txt

install: all
	export GOPATH=`cd ../../..;pwd`;\
	install -m 700 $$GOPATH/$(PKGPATH)/wisdom $(SBINDIR);\
	install -m 700 $$GOPATH/$(PKGPATH)/wisdomd $(SBINDIR)
