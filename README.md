# JuniorMayWork

Реальний час — трекер дрібних фриланс-замовлень (React/JavaScript, до $100). Go-скрейпер збирає замовлення з українських джерел, складає все в PostgreSQL, а Next.js UI показує їх деревом **вітки (ніші) → листочки (замовлення)** з повним CRUD над БД, заявками, звітами за датами і живим фідом подій по WebSocket. Зниклі або забрані замовлення виносяться в JSON-архів і видаляються з БД.

## Швидкий старт

Потрібно лише: **Go 1.25+** і **Node.js >= 20.9**. PostgreSQL ставити не треба — бекенд піднімає вбудований сам.

```bash
# 1. Backend: вбудований PostgreSQL + API + WebSocket + скрейпер на :8080
cd backend
go mod tidy
go run ./cmd/jmw

# 2. Frontend: UI на http://localhost:3000
cd frontend
npm install
npm run dev
```

Перший запуск бекенда завантажить бінарники вбудованого PostgreSQL (~40МБ) — далі старт миттєвий. База живе в `backend/data/pg` (каталог у `.gitignore`).

### Зовнішня база (опція)

Хочеш повноцінний PostgreSQL окремо — задай `JMW_DATABASE_URL`, наприклад:

```bash
set JMW_DATABASE_URL=postgres://postgres:ПАРОЛЬ@localhost:5432/jmw
go run ./cmd/jmw
```

## Як це працює

- **Скрейпер** (менеджер у `internal/scraper`) опитує джерела за `JMW_POLL_INTERVAL` (деф. 90s) і дифить результат проти БД: нове замовлення → `INSERT` + подія `new` + broadcast у WebSocket; зникле (не було останні 2 вибірки) → JSON-архів `backend/data/archive/orders-YYYY-MM.json` + `DELETE` з БД + подія `removed`. Виграні замовлення (`won`) не видаляються.
- **Вітки** (`branches`) — ваші ніші з ключовими словами та лімітом бюджету. Замовлення класифікується у вітку автоматично: найбільший збіг keywords за title+description, бюджет вкладається в ліміт.
- **Статуси замовлення**: `new → seen → applied → won / lost / archived`.
- **Заявки** (`applications`) — кнопка «подати заявку» фіксує спробу й результат (`pending / accepted / declined`), історія — на сторінці «Звіти» + живий фід показує, куди рухаються реквести.
- **Події WebSocket** несуть бюджет замовлення (`payload.budget_cents`), тож фронт одразу бачить суму; на гігі до $50 — тост + звук.

## Джерела (український фокус)

Кожне джерело — один файл у `backend/internal/scraper/sources/` під загальним реєстром `Sources()`:

- **telegram** — Telegram-канали через публічні веб-дзеркала `t.me/s/<канал>`. Список каналів — у `JMW_TELEGRAM_CHANNELS` (через кому, без `@`). Дефолт: `freelance_for_ukraine`. Бюджети в грн конвертуються у USD-центи (орієнтовний курс `uahPerUsd` у `common.go`).
- **freelancehunt** — офіційний API v2: `GET https://api.freelancehunt.com/v2/projects` (відкритий JSON:API). Токен опційний: `JMW_FREELANCEHUNT_TOKEN` (Профіль → API на freelancehunt.com) дає стабільніший доступ.
- **weblancer** — HTML weblancer.net/jobs через goquery (селектори захисні: якщо сайт змінить верстку, підкоригуй `weblancerCards` у `weblancer.go`).

Щоб додати новий сайт — реалізуй функцію `FetchMySite(ctx) ([]model.Listing, error)`, зареєструй її у `Sources()` і додай ім'я в `JMW_SOURCES`.

## Сторінки UI

| Сторінка | Що там |
|---|---|
| `/` | Дашборд: статистика, дерево вітка→листочок, живий фід подій, тости |
| `/orders` | Усі замовлення: фільтри (статус/вітка/пошук), зміна статусу і вітки, заявка, видалення |
| `/branches` | CRUD віток: назва, ключові слова, ліміт бюджету, активність |
| `/reports` | Звіти за діапазоном дат: бари по вітках/джерелах/днях, історія заявок з результатами, експорт CSV |
| `/settings` | Стан скрейпера, ручний запуск, шпаргалка конфігу |

## Структура

```
backend/
  cmd/jmw/main.go            точка входу (вбудована БД + API + WS + скрейпер)
  internal/config            env-конфіг з дефолтами
  internal/db                вбудований PostgreSQL, пул pgx, ранер міграцій
  internal/model             Branch/Order/Event/Application/Listing
  internal/store             усі SQL-запити (параметризовані)
  internal/scraper           менеджер: poll → diff → класифікація → архівація
  internal/scraper/sources   реєстр і реалізації джерел
  internal/archive           JSON-архів зниклих замовлень
  internal/ws                WebSocket-хаб
  internal/api               REST-хендлери, маршрути, CORS
  migrations/                SQL-міграції (вбудовані в бінарник)
  data/                      вбудована БД (pg) і JSON-архів (archive) — у .gitignore
frontend/
  app/                       сторінки Next.js (App Router)
  components/                TreeView, LeafCard, EventFeed, StatCard, NewOrderAlert
  lib/                       api-клієнт і useLive (WebSocket-хук)
  styles/                    globals.scss + BEM-блоки
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

- Усе, що назбирається, живе в PostgreSQL (таблиці `branches`, `orders`, `events`, `applications`) — за замовчуванням у вбудованій базі в `backend/data/pg`.
- Замовлення, зникле з джерела, перед видаленням із БД повністю (з `raw`-снапшотом) пишеться в `backend/data/archive/orders-YYYY-MM.json`.
- Історія подій (`events`) зберігає снапшот замовлення у `payload`, тому живе й після видалення рядка з `orders`.
- Бекап усього стану = скопіювати папку `backend/data`.
