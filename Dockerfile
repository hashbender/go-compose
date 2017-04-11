FROM golang

COPY . /go/src/github.com/nitronick600/go-compose

RUN go install github.com/nitronick600/go-compose/

# RUN apt-get update
# RUN add-apt-repository -y ppa:chris-lea/node.js
# RUN apt-get update
# RUN apt-get -y install nodejs

# ADD package.json /tmp/package.json
# RUN cd /tmp && npm install
# RUN mkdir -p /go/src/github.com/rippaio/rippa && cp -a /tmp/node_modules /go/src/github.com/rippaio/rippa

# RUN npm install

WORKDIR /go/src/github.com/nitronick600/go-compose
EXPOSE 5000