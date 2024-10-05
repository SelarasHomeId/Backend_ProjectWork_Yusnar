FROM golang:1.22.1-alpine as build

RUN mkdir /app

WORKDIR /app

COPY ./ /app

RUN go mod tidy

RUN go build -o selarashomeid

EXPOSE 80

CMD [ "./selarashomeid" ]