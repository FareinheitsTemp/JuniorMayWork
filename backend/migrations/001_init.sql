-- JuniorMayWork: початкова схема.
-- branches = "вітки" (ніші/МП), orders = "листочки" (замовлення).

CREATE TABLE IF NOT EXISTS branches (
  id               SERIAL PRIMARY KEY,
  name             TEXT NOT NULL UNIQUE,
  keywords         TEXT[] NOT NULL DEFAULT '{}',
  max_budget_cents INTEGER NOT NULL DEFAULT 10000,
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
  id            SERIAL PRIMARY KEY,
  source        TEXT NOT NULL,
  external_id   TEXT NOT NULL,
  url           TEXT NOT NULL DEFAULT '',
  title         TEXT NOT NULL,
  description   TEXT NOT NULL DEFAULT '',
  budget_cents  INTEGER,
  currency       TEXT NOT NULL DEFAULT 'USD',
  skills        TEXT[] NOT NULL DEFAULT '{}',
  branch_id     INTEGER REFERENCES branches(id) ON DELETE SET NULL,
  status        TEXT NOT NULL DEFAULT 'new'
                CHECK (status IN ('new','seen','applied','won','lost','archived')),
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  raw           JSONB,
  UNIQUE (source, external_id)
);

-- Історія подій: нові/оновлені/видалені замовлення, зміни статусів, заявки.
-- order_id = SET NULL, бо замовлення з БД видаляємо після архівації,
-- а снапшот (title, source, url) живе в payload.
CREATE TABLE IF NOT EXISTS events (
  id         SERIAL PRIMARY KEY,
  order_id   INTEGER REFERENCES orders(id) ON DELETE SET NULL,
  type       TEXT NOT NULL
             CHECK (type IN ('new','updated','removed','status_changed','applied')),
  payload    JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Заявки: "беру це замовлення" + результат.
-- order_id ON DELETE SET NULL + order_title: історія заявок живе
-- і після видалення замовлення з БД.
CREATE TABLE IF NOT EXISTS applications (
  id          SERIAL PRIMARY KEY,
  order_id    INTEGER REFERENCES orders(id) ON DELETE SET NULL,
  order_title TEXT NOT NULL DEFAULT '',
  note        TEXT NOT NULL DEFAULT '',
  result      TEXT NOT NULL DEFAULT 'pending'
              CHECK (result IN ('pending','accepted','declined')),
  applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_branch     ON orders(branch_id);
CREATE INDEX IF NOT EXISTS idx_orders_status     ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_first_seen ON orders(first_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_created    ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_apps_order        ON applications(order_id);
