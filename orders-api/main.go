package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "modernc.org/sqlite"
)

type Order struct {
	ID        int64   `json:"id"`
	UserID    int64   `json:"user_id"`
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Total     float64 `json:"total"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

var (
	db          *sql.DB
	usersURL    = env("USERS_API_URL", "http://users-api:8000")
	productsURL = env("PRODUCTS_API_URL", "http://products-api:8003")
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// fetch consulta otro microservicio; devuelve false si el recurso no existe
func fetch(url string, out any) (bool, error) {
	res, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return false, nil
	}
	return true, json.NewDecoder(res.Body).Decode(out)
}

const cols = "id, user_id, product_id, quantity, total, status, created_at"

func scan(row interface{ Scan(...any) error }) (Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.UserID, &o.ProductID, &o.Quantity, &o.Total, &o.Status, &o.CreatedAt)
	return o, err
}

func find(id string) (Order, error) {
	return scan(db.QueryRow("SELECT "+cols+" FROM orders WHERE id=?", id))
}

func main() {
	var err error
	if db, err = sql.Open("sqlite", env("DB_PATH", "/data/orders.db")); err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		total REAL NOT NULL,
		status TEXT NOT NULL DEFAULT 'pendiente',
		created_at TEXT DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var o Order
		if json.NewDecoder(r.Body).Decode(&o) != nil || o.UserID == 0 || o.ProductID == 0 || o.Quantity <= 0 {
			reply(w, 400, map[string]string{"error": "user_id, product_id y quantity son obligatorios"})
			return
		}
		// Comunicacion con Users API
		var user map[string]any
		if ok, err := fetch(fmt.Sprintf("%s/users/%d", usersURL, o.UserID), &user); err != nil || !ok {
			reply(w, 400, map[string]string{"error": "el usuario no existe en Users API"})
			return
		}
		// Comunicacion con Products API
		var product struct {
			Price float64 `json:"price"`
		}
		if ok, err := fetch(fmt.Sprintf("%s/products/%d", productsURL, o.ProductID), &product); err != nil || !ok {
			reply(w, 400, map[string]string{"error": "el producto no existe en Products API"})
			return
		}
		res, err := db.Exec("INSERT INTO orders (user_id, product_id, quantity, total) VALUES (?, ?, ?, ?)",
			o.UserID, o.ProductID, o.Quantity, product.Price*float64(o.Quantity))
		if err != nil {
			reply(w, 500, map[string]string{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		created, _ := find(strconv.FormatInt(id, 10))
		reply(w, 201, created)
	})

	http.HandleFunc("GET /orders", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT " + cols + " FROM orders")
		if err != nil {
			reply(w, 500, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()
		list := []Order{}
		for rows.Next() {
			if o, err := scan(rows); err == nil {
				list = append(list, o)
			}
		}
		reply(w, 200, list)
	})

	http.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		o, err := find(r.PathValue("id"))
		if err != nil {
			reply(w, 404, map[string]string{"error": "no encontrado"})
			return
		}
		reply(w, 200, o)
	})

	http.HandleFunc("PUT /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Status string `json:"status"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Status == "" {
			reply(w, 400, map[string]string{"error": "status es obligatorio"})
			return
		}
		res, _ := db.Exec("UPDATE orders SET status=? WHERE id=?", body.Status, r.PathValue("id"))
		if n, _ := res.RowsAffected(); n == 0 {
			reply(w, 404, map[string]string{"error": "no encontrado"})
			return
		}
		o, _ := find(r.PathValue("id"))
		reply(w, 200, o)
	})

	http.HandleFunc("DELETE /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		res, _ := db.Exec("DELETE FROM orders WHERE id=?", r.PathValue("id"))
		if n, _ := res.RowsAffected(); n == 0 {
			reply(w, 404, map[string]string{"error": "no encontrado"})
			return
		}
		reply(w, 200, map[string]string{"message": "pedido eliminado"})
	})

	log.Println("Orders API en puerto 8002")
	log.Fatal(http.ListenAndServe(":8002", nil))
}
