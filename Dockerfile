# Stage 1: builder
FROM maven:3.9-eclipse-temurin-21 AS builder

WORKDIR /build

COPY pom.xml ./
RUN mvn -q -B dependency:go-offline

COPY src ./src
RUN mvn -q -B package -DskipTests

# Stage 2: final
FROM eclipse-temurin:21-jre-alpine

RUN apk add --no-cache tzdata

COPY --from=builder /build/target/feature-management-service-*.jar /app/app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "/app/app.jar"]
