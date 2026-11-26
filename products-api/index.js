const http = require('node:http');
const { DatabaseSync } = require('node:sqlite');

const db = new DatabaseSync(process.env.DB_PATH || '/data/products.db');
db.exec(`CREATE TABLE IF NOT EXISTS products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  price REAL NOT NULL,
  description TEXT,
  stock INTEGER NOT NULL DEFAULT 0)`);

const find = (id) => db.prepare('SELECT * FROM products WHERE id = ?').get(id);

const send = (res, code, body) => {
  res.writeHead(code, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
};

const readBody = (req) => new Promise((resolve) => {
  let data = '';
  req.on('data', (c) => (data += c));
  req.on('end', () => { try { resolve(JSON.parse(data || '{}')); } catch { resolve({}); } });
});

http.createServer(async (req, res) => {
  const [, resource, id] = req.url.split('/');
  if (resource !== 'products') return send(res, 404, { error: 'ruta no encontrada' });

  if (!id && req.method === 'GET') return send(res, 200, db.prepare('SELECT * FROM products').all());

  if (!id && req.method === 'POST') {
    const b = await readBody(req);
    if (!b.name || b.price === undefined) return send(res, 400, { error: 'name y price son obligatorios' });
    const r = db.prepare('INSERT INTO products (name, price, description, stock) VALUES (?, ?, ?, ?)')
      .run(b.name, b.price, b.description ?? null, b.stock ?? 0);
    return send(res, 201, find(r.lastInsertRowid));
  }

  const p = id && find(id);
  if (!p) return send(res, 404, { error: 'no encontrado' });

  if (req.method === 'GET') return send(res, 200, p);

  if (req.method === 'PUT') {
    const b = { ...p, ...(await readBody(req)) };
    db.prepare('UPDATE products SET name = ?, price = ?, description = ?, stock = ? WHERE id = ?')
      .run(b.name, b.price, b.description, b.stock, id);
    return send(res, 200, find(id));
  }

  if (req.method === 'DELETE') {
    db.prepare('DELETE FROM products WHERE id = ?').run(id);
    return send(res, 200, { message: 'producto eliminado' });
  }

  send(res, 405, { error: 'metodo no permitido' });
}).listen(8003, () => console.log('Products API en puerto 8003'));
