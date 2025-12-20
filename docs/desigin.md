# Anti-Bruteforce — Software Design Description (SDD)

## 1. Введение
**Цель:**  
Документ описывает архитектуру, алгоритмы и технические решения микросервиса **Anti-Bruteforce**, реализующего защиту от подбора паролей при авторизации. 

**Общее описание**
Микросервис **Anti-Bruteforce** выполняет контроль количества повторов параметров авторизации (логин/пароль/IP) за заданный период времени.
Сервис ограничивает частоту попыток авторизации для различных комбинаций параметров, например:
* не более N = 10 попыток в минуту для данного логина.
* не более M = 100 попыток в минуту для данного пароля (защита от обратного brute-force).
* не более K = 1000 попыток в минуту для данного IP.

Для проверки лимитов по параметрам авторизации используется алгоритм Sliding Window, через механизм Redis ZSET.

**Основание:**  
Разработано в соответствии с требованиями, изложенными в документе  
[`docs/requirements.md`](requirements.md).

**Область применения:**  
Документ предназначен для разработчиков, ревьюеров и DevOps-инженеров, поддерживающих сервис.

---

## 2. Назначение и функции
Сервис вызывается перед авторизацией пользователя и возвращает:
- `ok=true` — если попытка разрешена;
- `ok=false` — если превышены лимиты или IP в blacklist.

**Функциональные возможности:**
- Проверка частоты попыток по логину, паролю и IP (gRPS-API)
- Проверка присутствия IP адреса в черном/белом списке подсетей(CIDR)
- Управление whitelist/blacklist подсетей (CIDR).
- Сброс счётчиков (bucket-ов).
- gRPC-API и CLI для администрирования.
- Конфигурирование через YAML/ENV.
- Сборка и запуск через Makefile и Docker Compose.

---

## 3. Архитектура системы
### 3.1 Структура каталогов

```
api/proto/anti_bruteforce/v1/               # proto файлы gRPC сервиса  

build/                  # Docker файлы       

cmd/
 ├─ anti-bruteforce/    # основной сервис (gRPC)
 ├─ abfctl/             # CLI-клиент для администрирования
 └─ migrator/           # мигратор (на goos)

configs/                # файл конфигурации

db/migrations/          # миграции

internal/
 ├─ abfclient/          # gRPC клиент 
 ├─ adapters/           # адаптеры для Sub/Pub операций
 ├─ app/                # бизнес-координация, use-cases
 ├─ config/             # парсинг YAML/env
 ├─ ctxmeta/            # работа с контекстом
 ├─ delivery/
 │  ├─grpc/             # реализация сервера
 │  └─interceptors/     # интерсепторы логирования и requestID
 ├─ domain/
 │   ├─ ratelimit/      # логика rate-limit
 │   └─ subnetlist/     # работа с white/black lists
 ├─ factory/            # фабрика (Redis client)
 ├─ integration/        # интеграционные тесты
 ├─ logger/             # slog + контекстные поля
 ├─ ports/              # набор портовых интерфейсов
 ├─ storage/            
 │   ├─ memory/         # in-memory реализация (тесты)
 │   ├─ postgresdb/     # хранение subnet lists
 │   └─ redisdb/        # реализация rate-limit в Redis (Sliding Window)
 └─ version/            # автоформирование версии
 

docs/
 ├─ architecture/      # архитектурные схемы (puml)
 ├─ requirements.md
 └─ design.md
```

---

## 4. Алгоритмы
### 4.1 Проверка попытки
1. Проверить IP в **whitelist** → разрешить.  
2. Проверить IP в **blacklist** → отказать.  
3. Проверить лимиты:  
   - `N` попыток/мин по логину  
   - `M` попыток/мин по паролю  
   - `K` попыток/мин по IP  
4. Если все три проверки в норме → `ok=true`.

### 4.2 Sliding Window (Redis ZSET)
Для ведения backets и проверки лимитов используется алгоритм Sliding Window, через Redis ZSET.
Атомарность операций обеспечивается путем работы в redis.Client.TxPipeline()

Используем паттерн "скользящее окно" на основе  Redis Sorted Set (ZSET)
с Member = текущее время в миллисекундах + случайное число.
В такой комбинации повторение Member маловероятно,
Считаем такую уникальность приемлимой, чтобы избежать коллизий при одновременных запросах.
TTL ключа устанавливаем в два раза больше длинны окна, чтобы данные не накапливались бесконечно.

Таким образом, по каждому ключу (например конкретному логину) в Redis создаётся отдельный ZSET,
в котором хранятся события обращения за проверкой — временные метки запросов.
Для проверки лимита - считаем количество элементов в ZSET, соответствующих текущему окну времени.
Если количество элементов меньше или равно лимиту - разрешаем действие.
Алгоритм:
1. Добавляем текущий запрос с текущей временной меткой.
2. Удаляем из множества все запросы старше текущего времени минус окно.
3. Считаем количество оставшихся запросов в множестве.
4. Если количество меньше или равно лимиту — разрешаем действие.

**Параметры: ZSET**  
- Key - строка, содержимое которой, это конкретный проверяемый элемент - логин, пароль(хеш) или IP адрес. например: "login:test_login"
- Score - текущее время в миллисекундах
- Member - текущее время в миллисекундах + случайное число

**Алгоритм:**
```text
// --- Создаём Redis пайплайн --- 
pipe := redis.Client.TxPipeline()
// 1. Добавляем текущий запрос
pipe.ZAdd( key, Score, Member)
// 2. Удаляем устаревшие
pipe.ZRemRangeByScore(key, "0", now-window)
// 3. Считаем, сколько элементов осталось в наборе
pipe.ZCard(key)
// 4. Обновляем TTL
pipe.Expire(key, window*2)
// ---- Запускаем пайплайн -----
pipe.Exec(ctx)
```

**Ключи:**
```
login:{login}
pass:{hash(password)}
ip:{ip}
```

**ResetBucket:** 
`DEL` соответствующего ключей.



### 4.3 Работа со списками
**Таблица PostgreSQL:**
```sql
CREATE TABLE subnets (
  CIDR TEXT NOT NULL,
  LIST_TYPE TEXT NOT NULL,
  COMMENT TEXT,
  DC TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (CIDR, LIST_TYPE)
);

```
**Предполагаются типы:** whitelist, blacklist.  
**Подсети:** IPv4, CIDR-нотация `192.1.1.0/25`.

---

## 5. API (gRPC)
### 5.1 Сервис `AntiBruteforce`
```proto
syntax = "proto3";
package abf.v1;

service AntiBruteforce {
  rpc CheckAttempt(CheckAttemptRequest) returns (CheckAttemptResponse);
  rpc ResetBucket(ResetBucketRequest) returns (ResetBucketResponse);
  rpc AddToWhitelist(ManageCIDRRequest) returns (ManageCIDRResponse);
  rpc RemoveFromWhitelist(ManageCIDRRequest) returns (ManageCIDRResponse);
  rpc AddToBlacklist(ManageCIDRRequest) returns (ManageCIDRResponse);
  rpc RemoveFromBlacklist(ManageCIDRRequest) returns (ManageCIDRResponse);
}
```

### 5.2 Сообщения
```proto
message CheckAttemptRequest {
  string login = 1;
  string password = 2;
  string ip = 3;
}
message CheckAttemptResponse { bool ok = 1; }

message ResetBucketRequest { string login = 1; string ip = 2; }
message ResetBucketResponse {}

message ManageCIDRRequest { string cidr = 1; }
message ManageCIDRResponse {}
```

---

## 6. CLI
Бинарник `cmd/abfctl`. Работает через gRPC API.

Примеры:
```bash
abfctl --addr 127.0.0.1:50051 check --login user --pass secret --ip 192.168.1.1
abfctl --addr 127.0.0.1:50051 blacklist add --cidr 192.168.2.0/24
abfctl --addr 127.0.0.1:50051 whitelist remove --cidr 192.168.1.0/24
abfctl --addr 127.0.0.1:50051 reset --login user  --ip 192.168.1.1
```

---

## 7. Безопасность
- Пароль не логируется, в Redis хранится хэш.  
- gRPC может работать поверх TLS.  
- CLI ограничен служебным доступом.

---

