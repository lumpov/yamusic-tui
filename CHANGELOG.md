# Changelog

Все значимые изменения проекта фиксируются в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/),
проект следует [Semantic Versioning](https://semver.org/lang/ru/).

## [Unreleased]

### Добавлено

- Sentry: DSN в `sentry-dsn` (`config.yaml`). Пустое значение — Sentry выключен. В issues уходят panic, `log.LVL_PANIC`, поломка JSON Yandex Music API и HTTP ≥ 500. INFO/WARN, 404 текстов, 400/401 не создают issue. Токен пишется как `02kdsjh...kjds8`. В событии: версия приложения, GOOS/GOARCH, Go, hostname, cache-tracks, lyrics, buffer, rewind ([#14](https://github.com/lumpov/yamusic-tui/pull/14)).

### Изменено

- Обложка трека не копится в `$TMPDIR/yamusic-tui/{id}.jpg`: каждый раз качается в один `cover.jpg`, файл живёт только пока играет текущий трек (MPRIS `file://`), при следующем треке и при выходе удаляются все `.jpg` в этом каталоге ([#12](https://github.com/lumpov/yamusic-tui/pull/12)).
- Перемотка по умолчанию: `shift+right` / `shift+left` вместо `ctrl+right` / `ctrl+left`. На macOS Control+←/→ забирает Mission Control (переключение рабочих столов), нажатие не доходит до TUI. Alt+стрелки забирают смену панелей в Windows Terminal и перемещение фокуса в оконных менеджерах, где Alt — основная модификаторная клавиша. Shift+стрелки на macOS, Linux и Windows доходят до приложения ([#7](https://github.com/lumpov/yamusic-tui/pull/7)).

### Исправлено

- Перемотка не переключает трек: позиция ограничена `[0, duration − шаг]`, `SeekEnd` не вызывается. Буфер стрима сохраняет хвост кадра при `UnexpectedEOF`, HTTP-тело закрывается всегда, `WriteTo` копирует снимок под `mux`. На Darwin после mid-track EOF feed продолжает кормить AudioQueue ([#7](https://github.com/lumpov/yamusic-tui/pull/7)).
- Darwin: `Close` дожидается выхода feed, затем `AudioQueueStop` и `AudioQueueDispose`. `AQPlayer` больше не освобождается по таймауту 500 мс, пока `aqParse` / колбэки очереди ещё могут работать с этим указателем ([#13](https://github.com/lumpov/yamusic-tui/pull/13)).
- Медиакнопки (MPRIS, SMTC, macOS Now Playing) больше не вызывают Play, Pause, Rewind, SetPos, SetVolume и перемешивание плейлиста из горутины обработчика. Команда уходит в `program.Send`, состояние плеера меняется в `Update` ([#9](https://github.com/lumpov/yamusic-tui/pull/9)).
- Плавное изменение громкости меняет плеер в `Update`, а не из горутины `volumeFader` ([#9](https://github.com/lumpov/yamusic-tui/pull/9)).
- Текст песни: после 404 LRC и TEXT запрашивается `GET /tracks/{id}/supplement` (`fullLyrics`) ([#5](https://github.com/lumpov/yamusic-tui/pull/5)).
- CI релиза: `macos-15` для darwin-arm64, `ubuntu-24.04`, Go из `go.mod`, повтор `go mod vendor` ([#5](https://github.com/lumpov/yamusic-tui/pull/5)).

## [0.9.0-beta.6] - 2026-10-28

### Безопасность

- `config.yaml` пишется и приводится к правам `0600` (только владелец), в том числе если файл уже существовал с более открытыми правами ([#4](https://github.com/lumpov/yamusic-tui/pull/4)).

### Исправлено

- Ошибка разбора YAML больше не перезаписывает `config.yaml` умолчаниями; токен со строки `token:` сохраняется, приложение не открывает экран входа ([#4](https://github.com/lumpov/yamusic-tui/pull/4)).
- Старый файл `token` читается через `TrimSpace` ([#4](https://github.com/lumpov/yamusic-tui/pull/4)).

## [0.9.0-beta.4] - 2026-10-28

### Изменено

- Заголовок плейера: «Yandex Music player», строка версии и пометка `unofficial` ([#3](https://github.com/lumpov/yamusic-tui/pull/3)).

### Исправлено

- Ошибки `json.Decoder.Decode` в ответах API и OAuth больше не маскируются пустым успехом ([#3](https://github.com/lumpov/yamusic-tui/pull/3)).
- `DurationMs` в `play-audio` и роторе уходит секундами (`DurationMs / 1000`), а не миллисекундами ([#3](https://github.com/lumpov/yamusic-tui/pull/3)).
- Числовые `label.id` и `lyricId` разбираются без ошибки JSON ([#3](https://github.com/lumpov/yamusic-tui/pull/3)).
- Текст песни: после 404 для LRC запрашивается TEXT; загрузка текста и обложки идёт через `httpClient`, тело ответа закрывается, не-200 отклоняется ([#3](https://github.com/lumpov/yamusic-tui/pull/3)).
- `TimeLimitedReader` запускает таймер простоя на первом `Read`, а не в конструкторе ([#3](https://github.com/lumpov/yamusic-tui/pull/3)).

## [0.9.0-beta.1] - 2026-10-27

### Добавлено

- На Darwin воспроизведение сжатого MP3 через AudioQueue, без декода в Go и без oto на горячем пути ([#1](https://github.com/lumpov/yamusic-tui/pull/1)).
- Теги с дефисом (`v0.9.0-beta.N`) публикуются как GitHub pre-release ([#1](https://github.com/lumpov/yamusic-tui/pull/1)).

### Изменено

- TUI не перерисовывает прогресс и красит неактивным текстом, пока терминал без фокуса ([#1](https://github.com/lumpov/yamusic-tui/pull/1)).
- Клавиши плейлистов: `+`/`-` вместо ctrl+up/down (на macOS эти сочетания не доходят до TUI) ([#1](https://github.com/lumpov/yamusic-tui/pull/1)).

### Исправлено

- Паники на пустых списках, nil-выборе, поиске плейлистов в `albumBlocks`, shuffle/ротор/лайки/кэш без элементов ([#2](https://github.com/lumpov/yamusic-tui/pull/2)).

[Unreleased]: https://github.com/lumpov/yamusic-tui/compare/v0.9.0-beta.6...HEAD
[0.9.0-beta.6]: https://github.com/lumpov/yamusic-tui/compare/v0.9.0-beta.4...v0.9.0-beta.6
[0.9.0-beta.4]: https://github.com/lumpov/yamusic-tui/compare/v0.9.0-beta.1...v0.9.0-beta.4
[0.9.0-beta.1]: https://github.com/lumpov/yamusic-tui/releases/tag/v0.9.0-beta.1
