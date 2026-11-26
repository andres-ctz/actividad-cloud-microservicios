package auth;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.sql.*;
import java.util.HashMap;
import java.util.Map;

public class Main {
    static final Gson GSON = new Gson();
    static final HttpClient HTTP = HttpClient.newHttpClient();
    static final String DB = "jdbc:sqlite:" + System.getenv().getOrDefault("DB_PATH", "/data/auth.db");
    static final String USERS_URL = System.getenv().getOrDefault("USERS_API_URL", "http://users-api:8000");

    record Login(String email, String password) {}

    public static void main(String[] args) throws Exception {
        try (Connection c = DriverManager.getConnection(DB); Statement s = c.createStatement()) {
            s.execute("CREATE TABLE IF NOT EXISTS login_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT, success INTEGER, created_at TEXT DEFAULT CURRENT_TIMESTAMP)");
        }
        HttpServer server = HttpServer.create(new InetSocketAddress(8001), 0);
        server.createContext("/health", ex -> send(ex, 200, Map.of("status", "ok", "service", "auth-api")));
        server.createContext("/login", Main::login);
        server.start();
        System.out.println("Auth API en puerto 8001");
    }

    static void login(HttpExchange ex) throws IOException {
        if (!ex.getRequestMethod().equals("POST")) { send(ex, 405, Map.of("error", "metodo no permitido")); return; }
        Login body = GSON.fromJson(new String(ex.getRequestBody().readAllBytes(), StandardCharsets.UTF_8), Login.class);
        if (body == null || body.email() == null || body.password() == null) {
            send(ex, 400, Map.of("error", "email y password son obligatorios")); return;
        }
        try {
            // Comunicacion con Users API: valida las credenciales
            HttpRequest req = HttpRequest.newBuilder(URI.create(USERS_URL + "/users/validate"))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(GSON.toJson(body))).build();
            JsonObject res = GSON.fromJson(HTTP.send(req, HttpResponse.BodyHandlers.ofString()).body(), JsonObject.class);
            boolean valid = res.get("valid").getAsBoolean();

            try (Connection c = DriverManager.getConnection(DB)) {
                PreparedStatement log = c.prepareStatement("INSERT INTO login_logs (email, success) VALUES (?, ?)");
                log.setString(1, body.email()); log.setInt(2, valid ? 1 : 0); log.executeUpdate();
            }
            Map<String, Object> out = new HashMap<>();
            out.put("valid", valid);
            out.put("message", valid ? "credenciales validas" : "credenciales invalidas");
            if (valid) out.put("user", res.get("user"));
            send(ex, valid ? 200 : 401, out);
        } catch (Exception e) {
            send(ex, 503, Map.of("error", "no se pudo contactar Users API: " + e.getMessage()));
        }
    }

    static void send(HttpExchange ex, int code, Object body) throws IOException {
        byte[] out = GSON.toJson(body).getBytes(StandardCharsets.UTF_8);
        ex.getResponseHeaders().set("Content-Type", "application/json");
        ex.sendResponseHeaders(code, out.length);
        ex.getResponseBody().write(out);
        ex.close();
    }
}
