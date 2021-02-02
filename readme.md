grab redis from docker: docker pull redis
run redis: docker run --name=redisInstance -p 6379:6379 redis
build the project: go build ./main.go
run it: ./main