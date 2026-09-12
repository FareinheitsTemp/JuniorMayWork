# JuniorMayWork

Реальний час — трекер дрібних фриланс-замовлень (React/JavaScript, до $100). Go-скрейпер збирає замовлення з публічних сайтів, складає все в PostgreSQL, а Next.js UI показує їх деревом **вітки (ніші) → листочки (замовлення)** з повним CRUD над БД, заявками, звітами за датами і живим фідом подій по WebSocket. Зниклі або забрані замовлення виносяться в JSON-архів і видаляються з БД.

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

Backend читає конфіг зі змінних оточення (`JMW_*`), дефолти збігаються з `.env.example`. У PowerShell змінні задаються так: `$env:JMW_DATABASE_URL="postgres://postgres:ПАРОЛЬ@localhost:5432/jmw"`.

## Як це працює

- **Скрейпер** (менеджер у `internal/scraper`) опитує джерела за `JMW_POLL_INTERVAL` (деф. 90s) і дифить результат проти БД: нове замовлення → `INSERT` + подія `new` + broadcast у WebSocket; зникле (не було останні 2 вибірки) → JSON-архів `backend/data/archive/orders-YYYY-MM.json` + `DELETE` з БД + подія `removed`. Виграні замовлення (`won`) не видаляються.
- **Вітки** (`branches`) — ваші ніші з ключовими словами та лімітом бюджету. Замовлення класифікується у вітку автоматично: найбільший збіг keywords за title+description, бюджет вкладається в ліміт.
- **Статуси замовлення**: `new → seen → applied → won / lost / archived`.
- **Заявки** (`applications`) — кнопка «подати заявку» фіксує спробу й результат (`pending / accepted / declined`), історія — на сторінці «Звіти» + живий фід показує, куди рухаються реквести.

## Сторінки UI

| Сторінка | Що там |
|---|---|
| `/` | Дашборд: статистика, дерево вітка→листочок, живий фід подій |
| `/orders` | Усі замовлення: фільтри (статус/вітка/пошук), зміна статусу і вітки, заявка, видалення |
| `/branches` | CRUD віток: назва, ключові слова, ліміт бюджету, активність |
| `/reports` | Звіти за діапазоном дат: бари по вітках/джерелах/днях, історія заявок з результатами, експорт CSV |
| `/settings` | Стан скрейпера, ручний запуск, шпаргалка конфігу |

## Джерела

Кожне джерело — один файл у `backend/internal/scraper/sources/` під загальним реєстром `Sources()`. Стартовий набір:

- **upwork** — RSS-пошук за запитами (react / javascript / next.js)
- **reddit** — r/slavelabour і r/forhire через публічний JSON
- **weblancer** — HTML weblancer.net/jobs через goquery (селектори захисні: якщо сайт змінить верстку, просто підкоригуй список `weblancerCards` у `weblancer.go`)

Щоб додати новий сайт — реалізуй функцію `FetchMySite(ctx) ([]model.Listing, error)`, зареєструй її у `Sources()` і додай ім'я в `JMW_SOURCES`.

## Структура

```
backend/
  cmd/jmw/main.go            точка входу (API + WS + скрейпер у фоні)
  internal/config            env-конфіг з дефолтами
  internal/db                пул pgx + ранер міграцій
  internal/model             Branch/Order/Event/Application/Listing
  internal/store             усі SQL-запити (параметризовані)
  internal/scraper           менеджер: poll → diff → класифікація → архівація
  internal/scraper/sources   реєстр і реалізації джерел
  internal/archive           JSON-архів зниклих замовлень
  internal/ws                WebSocket-хаб
  internal/api               REST-хендлери, маршрути, CORS
  migrations/                SQL-міграції (вбудовані в бінарник)
frontend/
  app/                       сторінки Next.js (App Router)
  components/                TreeView, LeafCard, EventFeed, StatCard
  lib/                       api-клієнт і useLive (WebSocket-хук)
  styles/                   globals.scss + BEM-блоки
```

## API (коротко)

| Метод | Шлях | Що робить |
|---|---|---|
| GET/POST | `/api/branches` | список / створення віток |
| PATCH/DELETE | `/api/branches/{id}` | редагування / видалення вітки |
| GET | `/api/orders` | список (фільтри: status, branch_id, q, from, to, limit) |
| GET/PATCH/DELETE | `/api/orders/{id}` | перегляд / редагування / видалення замовлення |
| POST | `/api/orders/{id}/apply` | подати заявку |
| GET | `/api/events` | лента подій |
| GET | `/api/applications` | історія заявок |
| PATCH | `/api/applications/{id}` | результат заявки (accepted/declined/pending) |
| GET | `/api/stats` | статистика дашборда |
| GET | `/api/reports/summary` | звіт за діапазоном дат (from, to у YYYY-MM-DD) |
| POST/GET | `/api/scraper/run` `/api/scraper/status` | керування збором |
| GET | `/ws` | WebSocket: події в реальному часі |

## Дані і архів

- Усе, що назбирається, живе в PostgreSQL (таблиці `branches`, `orders`, `events`, `applications`).
- Замовлення, зникле з джерела, перед видаленням із БД повністю (з `raw`-снапшотом) пишеться в `backend/data/archive/orders-YYYY-MM.json` — цей каталог у `.gitignore`.
- Історія подій (`events`) зберігає снапшот замовлення у `payload`, тому живе й після видалення рядка з `orders`.
