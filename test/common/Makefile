.PHONY: all

X86_64=x86_64
ARRCH64=aarch64
CC:=gcc
CFLAGS:=-Wall

all: sem net_test

sem:
	$(CC) $(CFLAGS) sem.c -o sem -lpthread

net_test:
	$(CC) $(CFLAGS) net_test.c -o net_test

clean:
	rm sem
	rm net_test
