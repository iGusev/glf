# GLF - GitLab Fuzzy Finder

<div align="center">
  <strong><a href="README.md">🇬🇧 English</a></strong> | <strong><a href="README_ru.md">🇷🇺 Русский</a></strong> | <strong><a href="README_cn.md">🇨🇳 中文</a></strong>
</div>

<br>

⚡ Быстрый CLI инструмент для мгновенного нечёткого поиска по проектам в self-hosted GitLab с использованием локального кэша.

<div align="center">
  <img src="demo.gif" alt="GLF Demo" />
</div>

[![CI](https://github.com/igusev/glf/workflows/CI/badge.svg)](https://github.com/igusev/glf/actions/workflows/ci.yml)
[![Security](https://github.com/igusev/glf/workflows/Security/badge.svg)](https://github.com/igusev/glf/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/igusev/glf/branch/main/graph/badge.svg)](https://codecov.io/gh/igusev/glf)
[![Go Report Card](https://goreportcard.com/badge/github.com/igusev/glf)](https://goreportcard.com/report/github.com/igusev/glf)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## ✨ Возможности

- ⚡ **Молниеносный нечёткий поиск** с локальным кэшированием
- 🔍 **Многотокенный поиск** - Поиск с пробелами: `"api storage"` найдёт проекты с обоими терминами
- 🧠 **Умное ранжирование** - Часто выбираемые проекты автоматически появляются первыми
- 🔁 **Автосинхронизация при запуске** - Проекты обновляются в фоне пока вы ищете
- 🔌 **JSON API режим** - Машиночитаемый вывод для Raycast, Alfred и собственных интеграций
- 🌍 **Кроссплатформенность** - Сборки для macOS, Linux и Windows

## 🚀 Быстрый старт

### Установка

#### Homebrew (macOS/Linux)

Самый простой способ установки GLF на macOS или Linux:

```bash
# Добавить tap
brew tap igusev/tap

# Установить GLF
brew install glf

# Обновить до последней версии
brew upgrade glf
```

#### MacPorts (macOS)

Альтернативный способ установки для пользователей macOS:

```bash
# Клонировать репозиторий портов
git clone https://github.com/igusev/macports-ports.git
cd macports-ports

# Добавить как локальный источник портов (требует sudo)
sudo bash -c "echo 'file://$(pwd)' >> /opt/local/etc/macports/sources.conf"

# Обновить и установить
sudo port sync
sudo port install glf

# Обновить до последней версии
sudo port selfupdate
sudo port upgrade glf
```

#### Scoop (Windows)

Самый простой способ установки GLF на Windows:

```powershell
# Добавить bucket
scoop bucket add igusev https://github.com/igusev/scoop-bucket

# Установить GLF
scoop install igusev/glf

# Обновить до последней версии
scoop update glf
```

#### Из исходников

```bash
# Клонировать репозиторий
git clone https://github.com/igusev/glf.git
cd glf

# Собрать и установить
make install
```

#### Бинарные релизы

Вы можете скачать официальные бинарные файлы GLF со [страницы релизов](https://github.com/igusev/glf/releases).

Доступно для: **macOS** (Intel & Apple Silicon), **Linux** (x64, ARM, ARM64 и др.), **Windows** (x64), **FreeBSD**, **OpenBSD**.

### Конфигурация

Запустите интерактивный мастер конфигурации:

```bash
glf --init
```

Он запросит:
- URL экземпляра GitLab (например, `https://gitlab.example.com`)
- Personal Access Token (с правами `read_api`)
- Таймаут API (по умолчанию: 30 секунд)

Конфигурация сохраняется в `~/.config/glf/config.yaml`.

Для сброса и переконфигурации:

```bash
glf --init --reset
```

#### Ручная конфигурация

Создайте `~/.config/glf/config.yaml`:

```yaml
gitlab:
  url: "https://gitlab.example.com"
  token: "your-personal-access-token"
  timeout: 30  # опционально, по умолчанию 30 секунд

cache:
  dir: "~/.cache/glf"  # опционально
```

#### Переменные окружения

Вы также можете использовать переменные окружения:

```bash
export GLF_GITLAB_URL="https://gitlab.example.com"
export GLF_GITLAB_TOKEN="your-token-here"
export GLF_GITLAB_TIMEOUT=30  # опционально
```

### Создание Personal Access Token

1. Перейдите в ваш экземпляр GitLab
2. Перейдите в **User Settings** → **Access Tokens**
3. Создайте новый токен с правами `read_api`
4. Скопируйте токен и используйте его в `glf --init`

### Синхронизация проектов

Получите проекты из GitLab и создайте локальный кэш:

```bash
glf sync
```

### Поиск проектов

#### Интерактивный режим (по умолчанию)

```bash
# Запустить интерактивный нечёткий поиск
glf

# Начать с начальным запросом
glf backend
```

**Навигация:**
- `↑/↓` - Навигация по результатам
- `Enter` - Выбрать проект
- `Ctrl+R` - Вручную обновить/синхронизировать проекты из GitLab
- `Ctrl+X` - Исключить/вернуть проект из результатов поиска
- `Ctrl+H` - Переключить отображение исключённых проектов
- `?` - Переключить текст помощи
- `Esc`/`Ctrl+C` - Выход
- Ввод текста для фильтрации проектов в реальном времени

**Индикатор активности:**
- `○` - Простой (ничего не происходит)
- `●` (зелёный) - Активен: синхронизация проектов или загрузка истории выборов
- `●` (красный) - Ошибка: синхронизация не удалась
- Автосинхронизация запускается при старте, ручная синхронизация доступна по `Ctrl+R`

## 📖 Использование

### Команды

```
glf [запрос]          Поиск проектов (по умолчанию: интерактивный TUI)
glf --init            Настроить подключение к GitLab
glf --init --reset    Сбросить и переконфигурировать подключение к GitLab
glf --sync            Синхронизировать проекты из GitLab в локальный кэш
glf --help            Показать справку
```

### Флаги

```
--init                Запустить интерактивный мастер конфигурации
--reset               Сбросить конфигурацию и начать с нуля (использовать с --init)
-g, --open            Алиас для --go (для совместимости)
--go                  Автоматически выбрать первый результат и открыть в браузере
-s, --sync            Синхронизировать кэш проектов
--full                Принудительная полная синхронизация (использовать с --sync)
-v, --verbose         Включить подробное логирование
--scores              Показать разбивку очков для отладки ранжирования
--json                Вывести результаты в формате JSON (для API интеграций)
--limit N             Ограничить количество результатов в JSON режиме (по умолчанию: 20)
```

### Примеры

```bash
# Интерактивный поиск
glf

# Поиск с предзаполненным запросом
glf microservice

# Многотокенный поиск (соответствие проектов со всеми терминами)
glf api storage        # Найдёт проекты содержащие и "api" И "storage"
glf user auth service  # Найдёт проекты со всеми тремя терминами

# Автоматически выбрать первый результат и открыть в браузере
glf ingress -g         # Откроет первое совпадение с "ingress"
glf api --go           # То же что и -g (алиас для совместимости)

# Открыть текущий Git репозиторий в браузере
glf .

# Синхронизировать проекты из GitLab
glf --sync             # Инкрементальная синхронизация
glf --sync --full      # Полная синхронизация (удаляет удалённые проекты)

# Подробный режим для отладки
glf sync --verbose

# Показать очки ранжирования для отладки
glf --scores

# Настроить подключение к GitLab
glf --init

# Сбросить и переконфигурировать
glf --init --reset
```

### Режим JSON вывода (API интеграция)

GLF поддерживает JSON вывод для интеграции с инструментами типа Raycast, Alfred или собственными скриптами:

```bash
# Вывести результаты поиска в JSON
glf --json api

# Ограничить количество результатов
glf --json --limit 5 backend

# Включить очки релевантности (опционально)
glf --json --scores microservice

# Получить все проекты (без запроса)
glf --json --limit 100
```

**Формат JSON вывода (без --scores):**

```json
{
  "query": "api",
  "results": [
    {
      "path": "backend/api-server",
      "name": "API Server",
      "description": "REST API для аутентификации",
      "url": "https://gitlab.example.com/backend/api-server"
    }
  ],
  "total": 5,
  "limit": 20
}
```

**Формат JSON вывода (с --scores):**

```json
{
  "query": "api",
  "results": [
    {
      "path": "backend/api-server",
      "name": "API Server",
      "description": "REST API для аутентификации",
      "url": "https://gitlab.example.com/backend/api-server",
      "score": 123.45
    }
  ],
  "total": 5,
  "limit": 20
}
```

**Разбивка очков:**

При использовании флага `--scores`, каждый проект включает очки релевантности, которые комбинируют:
- **Релевантность поиска**: Нечёткое совпадение + очки полнотекстового поиска
- **История использования**: Частота предыдущих выборов (с экспоненциальным затуханием)
- **Бустинг для конкретного запроса**: 3x множитель для проектов, выбранных с этим точным запросом

Более высокие очки указывают на лучшие совпадения. Проекты автоматически сортируются по очкам (по убыванию).

**Примеры использования:**
- **Расширение для Raycast**: Быстрая навигация по проектам из Raycast
- **Workflow для Alfred**: Поиск проектов GitLab в Alfred
- **CI/CD скрипты**: Автоматическое обнаружение проектов и генерация URL
- **Собственные инструменты**: Создавайте собственные интеграции на основе поиска GLF
- **Аналитика**: Используйте `--scores` для понимания ранжирования и оптимизации поисковых запросов

**Обработка ошибок:**

При возникновении ошибок, GLF выводит JSON формат ошибки и завершается с кодом 1:

```json
{
  "error": "no projects in cache"
}
```

### Умное ранжирование

GLF изучает ваши паттерны выбора и автоматически повышает часто используемые проекты:

- **Первый раз**: Поиск `"api"` → Выбор `myorg/api/storage`
- **В следующий раз**: Поиск `"api"` → `myorg/api/storage` появляется **первым**!
- Чем чаще вы выбираете проект, тем выше он ранжируется
- Бустинг для конкретного запроса: проекты, выбранные для конкретных поисковых терминов, ранжируются выше для этих терминов
- Недавние выборы получают дополнительный буст (последние 7 дней)

История хранится в `~/.cache/glf/history.gob` и сохраняется между сессиями.

## 🔧 Разработка

### Сборка

```bash
# Собрать для текущей платформы
make build

# Собрать для всех платформ
make build-all

# Собрать для конкретной платформы
make build-linux
make build-macos
make build-windows

# Создать архивы релиза
make release
```

### Тестирование

```bash
# Запустить тесты
make test

# Запустить тесты с покрытием
make test-coverage

# Форматировать код
make fmt

# Запустить линтеры
make lint
```

### Релизы

GLF использует автоматизированный CI/CD для релизов через GitHub Actions и [GoReleaser](https://goreleaser.com/).

#### Автоматический процесс релиза

При пуше нового тега версии, workflow релиза автоматически:

1. ✅ Собирает бинарники для всех поддерживаемых платформ (macOS, Linux, Windows, FreeBSD, OpenBSD)
2. ✅ Создаёт GitHub Release с артефактами и changelog
3. ✅ Обновляет [Homebrew tap](https://github.com/igusev/homebrew-tap) для пользователей macOS/Linux
4. ✅ Обновляет [MacPorts Portfile](https://github.com/igusev/macports-ports) для пользователей macOS
5. ✅ Обновляет [Scoop bucket](https://github.com/igusev/scoop-bucket) для пользователей Windows

#### Создание нового релиза

```bash
# Создать и запушить тег версии
git tag v0.3.0
git push origin v0.3.0

# GitHub Actions автоматически:
# - Запустит GoReleaser
# - Соберёт кроссплатформенные бинарники
# - Создаст GitHub release
# - Обновит пакетные менеджеры (Homebrew, MacPorts, Scoop)
```

#### Ручной релиз (опционально)

Вы также можете запустить релизы вручную из UI GitHub Actions:
- Перейдите в **Actions** → **Release** → **Run workflow**

### Структура проекта

```
glf/
├── cmd/glf/              # Точка входа CLI
│   └── main.go           # Главная команда и логика поиска
├── internal/
│   ├── config/           # Обработка конфигурации
│   ├── gitlab/           # GitLab API клиент
│   ├── history/          # Отслеживание частоты выборов
│   ├── index/            # Индексирование описаний (Bleve)
│   ├── logger/           # Утилиты логирования
│   ├── search/           # Комбинированный нечёткий + полнотекстовый поиск
│   ├── sync/             # Логика синхронизации
│   ├── tui/              # Terminal UI (Bubbletea)
│   └── types/            # Общие типы
├── Makefile              # Автоматизация сборки
└── README.md
```

## ⚙️ Опции конфигурации

### Настройки GitLab

| Опция | Описание | По умолчанию | Обязательно |
|-------|----------|--------------|-------------|
| `gitlab.url` | URL экземпляра GitLab | - | Да |
| `gitlab.token` | Personal Access Token | - | Да |
| `gitlab.timeout` | Таймаут API в секундах | 30 | Нет |

### Настройки кэша

| Опция | Описание | По умолчанию | Обязательно |
|-------|----------|--------------|-------------|
| `cache.dir` | Путь к директории кэша | `~/.cache/glf` | Нет |

### Исключения

| Опция | Описание | По умолчанию | Обязательно |
|-------|----------|--------------|-------------|
| `exclusions` | Список путей проектов для исключения | `[]` | Нет |

Пример с исключениями:

```yaml
gitlab:
  url: "https://gitlab.example.com"
  token: "your-token"

exclusions:
  - "archived/old-project"
  - "deprecated/legacy-api"
```

Исключённые проекты можно переключать с помощью `Ctrl+X` в TUI или скрывать/показывать с помощью `Ctrl+H`.

## 🐛 Устранение неполадок

### Проблемы с подключением

```bash
# Используйте подробный режим для просмотра детальных логов
glf sync --verbose
```

**Распространённые проблемы:**
- Неверный URL GitLab: Проверьте URL в конфиге
- Токен истёк: Перегенерируйте токен в GitLab
- Таймаут сети: Увеличьте таймаут в конфиге
- Недостаточно прав: Убедитесь что токен имеет права `read_api`

### Проблемы с кэшем

```bash
# Проверить местоположение кэша
ls -la ~/.cache/glf/

# Очистить кэш и пересинхронизировать
rm -rf ~/.cache/glf/
glf sync
```

### Проблемы с конфигурацией

```bash
# Переконфигурировать подключение к GitLab
glf --init

# Сбросить и переконфигурировать с нуля
glf --init --reset

# Проверить текущую конфигурацию
cat ~/.config/glf/config.yaml
```

## 📝 Лицензия

Лицензия MIT - см. файл [LICENSE](LICENSE) для деталей.

## 🤝 Вклад

Вклад приветствуется! Не стесняйтесь отправлять issues и pull requests.

## 🙏 Благодарности

- Собрано с [Cobra](https://github.com/spf13/cobra) для фреймворка CLI
- UI на [Bubbletea](https://github.com/charmbracelet/bubbletea)
- Стилизация с [Lipgloss](https://github.com/charmbracelet/lipgloss)
- Индексирование поиска с [Bleve](https://github.com/blevesearch/bleve)
- GitLab API через [go-gitlab](https://gitlab.com/gitlab-org/api/client-go)
