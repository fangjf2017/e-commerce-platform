.PHONY: build test docker-up docker-down run package

build:
	mvn -q -B compile

package:
	mvn -q -B package -DskipTests

test:
	mvn -q -B test

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run:
	mvn -q spring-boot:run
