FROM registry.semaphoreci.com/golang:1.18 as builder

ENV APP_HOME /go/src/go-chat-server
WORKDIR "$APP_HOME"

COPY . .

RUN go mod download
RUN go mod verify
RUN go build -o server


FROM registry.semaphoreci.com/golang:1.18

ENV APP_HOME /go/src/go-chat-server
RUN mkdir -p "$APP_HOME"
WORKDIR "$APP_HOME"

COPY --from=builder "$APP_HOME"/server $APP_HOME
COPY --from=builder "$APP_HOME"/.dev $APP_HOME
COPY --from=builder "$APP_HOME"/.prod $APP_HOME

EXPOSE 8080
CMD ["./server"]