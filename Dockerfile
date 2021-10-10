FROM golang:alpine

RUN mkdir /doremy
ADD . /doremy/

WORKDIR /doremy

RUN go build -o main .
RUN adduser -S -D -H -h /doremy doremy
USER doremy

CMD ["./main"]
