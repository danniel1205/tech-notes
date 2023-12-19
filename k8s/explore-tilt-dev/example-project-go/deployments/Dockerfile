FROM golang:1.18
WORKDIR /app
ADD . .
RUN go mod tidy
RUN go install ./
ENTRYPOINT tilt-dev