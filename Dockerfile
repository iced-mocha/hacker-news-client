FROM golang:1.9

RUN go get -u github.com/golang/dep/cmd/dep && go install github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/icedmocha/hacker-news-client
COPY . /go/src/github.com/icedmocha/hacker-news-client

RUN dep ensure && go install

ENTRYPOINT ["hacker-news-client"]
