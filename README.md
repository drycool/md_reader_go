# MD Reader (Go Edition) 📖

![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20WSL-blue)

**MD Reader** — это мощный и легкий инструмент для работы с Markdown-документацией. Он объединяет в себе скорость консольной утилиты и удобство современного графического интерфейса. 

Проект идеально подходит для разработчиков, которые ведут базу знаний в `.md` файлах (например, Obsidian, Logseq или GitHub Wiki) и хотят иметь быстрый доступ к ней без тяжелых редакторов.

---

## 🚀 Основные возможности

### 🖥 Графический интерфейс (GUI)
- **Sidebar Navigation**: Автоматическое дерево оглавления для всех файлов в папке.
- **Живой поиск**: Нечеткий поиск (Fuzzy Search) по заголовкам в реальном времени.
- **Markdown Rendering**: Красивое отображение текста с поддержкой форматирования.
- **Портативность**: Запуск в одно окно без лишних консолей.

### 💻 Консольный интерфейс (CLI)
- **Interactive Mode**: Поиск и чтение разделов прямо в терминале.
- **TOC Generator**: Вывод структуры любого Markdown файла.
- **Statistics**: Анализ объема документации (строки, файлы, заголовки).

---

## 🛠 Установка и сборка

### Требования
Для сборки графической версии (Fyne) требуется C-компилятор (**GCC/MinGW**):
- **Windows**: [w64devkit](https://github.com/skeeto/w64devkit) или MSYS2.
- **WSL/Ubuntu**: `sudo apt install libgl1-mesa-dev xorg-dev`.

### Сборка
```powershell
# Сборка GUI-версии (без консольного окна)
go build -ldflags="-H windowsgui" -o "MD Reader.exe" ./cmd/md-reader

# Сборка обычной CLI-версии
go build -o md-reader.exe ./cmd/md-reader
```

---

## 📂 Структура проекта

| Папка | Описание |
|-------|----------|
| `cmd/` | Точки входа в приложение |
| `internal/gui/` | Логика Fyne интерфейса |
| `internal/viewer/` | Ядро поиска и CLI-отображения |
| `internal/toc/` | Парсер структуры Markdown |
| `internal/loader/` | Безопасная загрузка файлов и кодировок |

---

## 📝 Использование

### Запуск GUI
Просто запустите файл или используйте команду:
```powershell
.\md-reader.exe gui .
```

### Команды CLI
- `open [path]` — открыть файл/папку в терминале.
- `toc [path]` — показать оглавление.
- `stats [path]` — статистика по документации.

---

## 🤝 Контрибьютинг

Приветствуются любые Pull Request! Особенно интересны:
- Поддержка Mermaid диаграмм.
- Темная/светлая темы для GUI.
- Экспорт в PDF/HTML.

---
*Разработано с использованием Go и Fyne.*
