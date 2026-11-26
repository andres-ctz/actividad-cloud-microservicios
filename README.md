# Microservicios multilenguaje con SQLite y Docker

Cuatro APIs independientes, cada una con su propia base de datos SQLite persistida en un volumen de Docker.

| API          | Lenguaje             | Puerto | Base de datos | Volumen         |
|--------------|----------------------|--------|---------------|-----------------|
| Users API    | Python (Flask)       | 8000   | users.db      | users-data      |
| Auth API     | Java 21 (HttpServer) | 8001   | auth.db       | auth-data       |
| Orders API   | Go (net/http)        | 8002   | orders.db     | orders-data     |
| Products API | JavaScript (Node 24) | 8003   | products.db   | products-data   |

```
.
├── docker-compose.yml
├── users-api/     app.py, requirements.txt, Dockerfile
├── auth-api/      src/main/java/auth/Main.java, pom.xml, Dockerfile
├── orders-api/    main.go, go.mod, go.sum, Dockerfile
└── products-api/  index.js, package.json, Dockerfile
```

## Comunicación entre servicios

Los cuatro contenedores comparten la red `microservices` definida en `docker-compose.yml` y se llaman entre sí por el nombre del servicio (DNS interno de Docker). Las URLs se pasan como variables de entorno.

```
Auth API  ──POST /users/validate──▶ Users API
Orders API ──GET /users/{id}──────▶ Users API
Orders API ──GET /products/{id}───▶ Products API
```

- **Auth → Users:** el login valida email y contraseña contra Users API y registra el intento en `auth.db`.
- **Orders → Users + Products:** al crear un pedido se verifica que el usuario y el producto existan, y el total se calcula con el precio de Products API.

Cada servicio sigue accediendo **solo a su propia base de datos**; los datos de otros servicios se obtienen únicamente por HTTP.

Cada API crea su base de datos y sus tablas al arrancar (`CREATE TABLE IF NOT EXISTS`) en `/data`, que es el volumen exclusivo de su contenedor.

| Base de datos | Tablas |
|---------------|--------|
| users.db      | `users(id, name, email, password)` |
| auth.db       | `login_logs(id, email, success, created_at)` |
| orders.db     | `orders(id, user_id, product_id, quantity, total, status, created_at)` |
| products.db   | `products(id, name, price, description, stock)` |

## Ejecución

```bash
docker compose up -d --build   # construir y levantar
docker compose ps              # estado de los contenedores
docker compose logs -f         # logs (o: docker compose logs users-api)
docker compose down            # detener y eliminar contenedores (los volúmenes se conservan)
docker compose down -v         # eliminar también los volúmenes (borra los datos)
```

## Endpoints

### Users API – :8000
```bash
curl -X POST localhost:8000/users -H "Content-Type: application/json" -d '{"name":"Ana","email":"ana@mail.com","password":"123"}'
curl localhost:8000/users
curl localhost:8000/users/1
curl -X PUT localhost:8000/users/1 -H "Content-Type: application/json" -d '{"name":"Ana Maria"}'
curl -X POST localhost:8000/users/validate -H "Content-Type: application/json" -d '{"email":"ana@mail.com","password":"123"}'   # usado por Auth API
curl -X DELETE localhost:8000/users/1
```

### Auth API – :8001
Valida las credenciales contra Users API (crea antes un usuario). Cada intento se registra en la tabla `login_logs`.
```bash
curl localhost:8001/health
curl -X POST localhost:8001/login -H "Content-Type: application/json" -d '{"email":"ana@mail.com","password":"123"}'   # valid: true
curl -X POST localhost:8001/login -H "Content-Type: application/json" -d '{"email":"ana@mail.com","password":"otra"}'  # valid: false (401)
```

### Orders API – :8002
Requiere que existan el usuario (Users API) y el producto (Products API).
```bash
curl -X POST localhost:8002/orders -H "Content-Type: application/json" -d '{"user_id":1,"product_id":1,"quantity":3}'   # total = precio * cantidad
curl localhost:8002/orders
curl localhost:8002/orders/1
curl -X PUT localhost:8002/orders/1 -H "Content-Type: application/json" -d '{"status":"enviado"}'
curl -X DELETE localhost:8002/orders/1
```

### Products API – :8003
```bash
curl -X POST localhost:8003/products -H "Content-Type: application/json" -d '{"name":"Mouse","price":25.5,"description":"Inalambrico","stock":10}'
curl localhost:8003/products
curl localhost:8003/products/1
curl -X PUT localhost:8003/products/1 -H "Content-Type: application/json" -d '{"stock":7}'
curl -X DELETE localhost:8003/products/1
```

## Prueba de persistencia

```bash
curl -X POST localhost:8000/users -H "Content-Type: application/json" -d '{"name":"Luis","email":"luis@mail.com","password":"abc"}'
docker compose down
docker compose up -d
curl localhost:8000/users                # el usuario sigue existiendo
docker network ls                        # actividadcloud_microservices
docker volume ls                         # actividadcloud_users-data, auth-data, orders-data, products-data
docker compose exec users-api ls /data   # users.db
```
