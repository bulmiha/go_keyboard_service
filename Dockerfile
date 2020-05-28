FROM golang
WORKDIR /app
RUN go get github.com/markbates/pkger/cmd/pkger
RUN go install -v github.com/markbates/pkger/cmd/pkger
COPY . /app
RUN pkger
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /app/app .
ENTRYPOINT  ["./app"]  