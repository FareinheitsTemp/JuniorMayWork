# JuniorMayWork

Реальний час — трекер дрібних фриланс-замовлень (React/JavaScript, до $100). Go-скрейпер збирає замовлення з публічних сайтів, складає все в PostgreSQL, а Next.js UI показує їх деревом **вітки (ніші) → листочки (замовлення)** з повним CRUD над БД, заявками, звітами за датами і живим фідом подій по WebSocket. Зниклі або зайняті замовлення виносяться в JSON-архів і видаляються з БД.

## Швидкий старт (без Docker)

Потрібно: локальний PostgreSQL 16+, Go 1.25+, Node.js >= 20.9.

```bash
# 1. База даних (один раз) — створи порожню базу
psql -U postgres -c "CREATE DATABASE jmw;"

# 2. Конфіг: скопіюй .env.example в .env і впиши свій пароль у JMW_DATABASE_URL
cp .env.example .env

# 3. Backend: API + WebSocket + скрейпер на :8080 (міграції застосуються самі)
cd backend
go mod tidy
go run ./cmd/jmw

# 4. Frontend: UI на http://localhost:3000
cd frontend
npm install
npm run dev
```

Backend читає конфіг зі змінних оточення (`JMW_*`), дефолти збігаються з `.env.example`.

## Як це працює

- **Скрейпер** (менеджер у `internal/scraper`) опитує джерела за `JMW_POLL_INTERVAL` (деф. 90s), дифить результат проти БД: нове замовлення → `INSERT` + подія `new` + broadcast у WebSocket; зникле (не було останні 2 вибірки) → JSON-архів `data/archive/orders-YYYY-MM.json` + `DELETE` з БД + подія `removed`. Виграні замовлення (`won`) не видаляються.
- **Вітки** (`branches`) — ваші ніші з ключовими словами та лімітом бюджету. Замовлення класифікується у вітку автоматично за `keywords`.
- **Статуси замовлення**: `new → seen → applied → won / lost / archived`.
- **Заявки** (`applications`) — кнопка «подати заявку» фіксує спробу й результат (`pending / accepted / declined`), історія подій показує, куди рухаються реквести.

## Джерела

Кожне джерело — один файл у `backend/internal/scraper/sources/` під інтерфейсом `Source`. Стартовий набір: Upwork (RSS-пошук), Reddit (`r/slavelabour`, `r/forhire`), Weblancer. Щоб додати сайт — реалізуйте `Fetch(ctx) ([]Listing, error)` і додайте ім'я в `JMW_SOURCES`.

## Структура

```
backend/   Go: cmd/jmw, internal/{config,db,model,store,scraper,archive,ws,api}, migrations/
frontend/  Next.js (App Router): app/, components/, lib/, styles/ (SCSS + BEM)
```

## API (коротко)

| Метод | Шлях | Що робить |
|---|---|---|
| GET/POST | `/api/branches` | список / створення віток |
| PATCH/DELETE | `/api/branches/{id}` | редагування / видалення вітки |
| GET | `/api/orders` | список (фільтри: status, branch_id, q, from, to) |
| PATCH/DELETE | `/api/orders/{id}` | зміна статусу / видалення замовлення |
| POST | `/api/orders/{id}/apply` | подати заявку |
| GET | `/api/events` | лента подій |
| GET | `/api/applications` | історія заявок |
| GET | `/api/reports/summary` | звіт за діапазоном дат |
| POST | `/api/scraper/run` | запустити збір негайно |
| GET | `/ws` | WebSocket: події в реальному часі |
