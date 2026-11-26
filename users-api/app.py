import os, sqlite3
from flask import Flask, request, jsonify

DB = os.getenv("DB_PATH", "/data/users.db")
app = Flask(__name__)

def db():
    conn = sqlite3.connect(DB)
    conn.row_factory = sqlite3.Row
    return conn

with db() as c:
    c.execute("""CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL)""")

def public(row):
    return {"id": row["id"], "name": row["name"], "email": row["email"]}

@app.post("/users")
def create():
    d = request.get_json(force=True, silent=True) or {}
    if not all(k in d for k in ("name", "email", "password")):
        return jsonify(error="name, email y password son obligatorios"), 400
    with db() as c:
        try:
            cur = c.execute("INSERT INTO users (name, email, password) VALUES (?, ?, ?)",
                            (d["name"], d["email"], d["password"]))
        except sqlite3.IntegrityError:
            return jsonify(error="email ya registrado"), 409
        return jsonify(public(c.execute("SELECT * FROM users WHERE id=?", (cur.lastrowid,)).fetchone())), 201

@app.post("/users/validate")
def validate():
    d = request.get_json(force=True, silent=True) or {}
    with db() as c:
        r = c.execute("SELECT * FROM users WHERE email=? AND password=?",
                      (d.get("email"), d.get("password"))).fetchone()
    return jsonify(valid=bool(r), user=public(r) if r else None)

@app.get("/users")
def list_all():
    with db() as c:
        return jsonify([public(r) for r in c.execute("SELECT * FROM users")])

@app.get("/users/<int:uid>")
def get_one(uid):
    with db() as c:
        r = c.execute("SELECT * FROM users WHERE id=?", (uid,)).fetchone()
    return jsonify(public(r)) if r else (jsonify(error="no encontrado"), 404)

@app.put("/users/<int:uid>")
def update(uid):
    d = request.get_json(force=True, silent=True) or {}
    with db() as c:
        r = c.execute("SELECT * FROM users WHERE id=?", (uid,)).fetchone()
        if not r:
            return jsonify(error="no encontrado"), 404
        c.execute("UPDATE users SET name=?, email=?, password=? WHERE id=?",
                  (d.get("name", r["name"]), d.get("email", r["email"]), d.get("password", r["password"]), uid))
        return jsonify(public(c.execute("SELECT * FROM users WHERE id=?", (uid,)).fetchone()))

@app.delete("/users/<int:uid>")
def delete(uid):
    with db() as c:
        n = c.execute("DELETE FROM users WHERE id=?", (uid,)).rowcount
    return jsonify(message="usuario eliminado") if n else (jsonify(error="no encontrado"), 404)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000)
