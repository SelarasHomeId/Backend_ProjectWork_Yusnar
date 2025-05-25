FROM golang:1.24.3-alpine as build

RUN mkdir /app

WORKDIR /app

COPY ./ /app

RUN go mod tidy

RUN go build -o selarashomeid

EXPOSE 80

CMD [ "./selarashomeid" ]