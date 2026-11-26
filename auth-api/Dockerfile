FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /app
COPY pom.xml .
RUN mvn -q dependency:go-offline
COPY src ./src
RUN mvn -q package

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /app/target/auth-api.jar app.jar
ENV DB_PATH=/data/auth.db
RUN mkdir -p /data
VOLUME /data
EXPOSE 8001
CMD ["java", "-jar", "app.jar"]
