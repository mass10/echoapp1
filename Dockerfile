# FROM ubuntu:latest
FROM golang:1.13

MAINTAINER mass10

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get upgrade

COPY app /app

WORKDIR /app

CMD go run main.go --release
