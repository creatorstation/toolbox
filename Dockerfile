FROM ubuntu:20.04

WORKDIR /app

RUN apt-get update
RUN apt-get install -y ca-certificates
RUN apt-get install -y ffmpeg
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

COPY bin .

ENTRYPOINT ["./bootstrap"]

EXPOSE 8080