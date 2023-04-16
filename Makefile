.PHONY: clean build package

all: build

RM		=	rm -rf
MKD		=	mkdir -p
TAR		=	tar
GO		=	go

NAME	=	instllr
BIN		=	./bin

GO_SRC	=	$(shell find ./ -type f -name '*.go')

clean:
	$(RM) $(BIN)

build: $(BIN) $(BIN)/$(NAME)

package: build
	$(eval version := $(shell cat VERSION))
	$(TAR) cfJ $(NAME)-${version}.tar.xz $(BIN)/$(NAME) LICENSE README.md

$(BIN):
	$(MKD) $(BIN)

$(BIN)/$(NAME): $(GO_SRC)
	CGO_ENABLED=0 $(GO) build -o $(BIN)/$(NAME) $(GO_SRC)
