# Changelog

Все значимые изменения проекта фиксируются в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/),
проект следует [Semantic Versioning](https://semver.org/lang/ru/).

## [Unreleased]

### Добавлено

- `playbacktrace`: события воспроизведения в памяти, сброс в `$TMPDIR/yamusic-tui/yamusic-tui.log` при смене трека и выходе. По умолчанию выключен (`Enabled = false`) ([#23](https://github.com/lumpov/yamusic-tui/pull/23)).
- Sentry: DSN в `sentry-dsn` (`config.yaml`). Пустое значение — Sentry выключен. В issues уходят panic, `log.LVL_PANIC`, поломка JSON Yandex Music API и HTTP ≥ 500. INFO/WARN, 404 текстов, 400/401 не создают issue. Токен пишется как `02kdsjh...kjds8`. В событии: версия приложения, GOOS/GOARCH, Go, hostname, cache-tracks, lyrics, buffer, rewind ([#14](https://github.com/lumpov/yamusic-tui/pull/14)).

### Безопасность

- `config.Save` не записывает пустой токен поверх существующего в `config.yaml`. `InitTest` перед каждым тестом копирует оригинал в свой каталог: yaml плеера на машине запуска (`~/.config/yamusic-tui/config.yaml` на момент старта процесса), если файла нет — `config/testdata/config.yaml`. Экран входа не сохраняет пустую строку.
- Каталоги приложения (`~/.config/yamusic-tui`, кэш, `$TMPDIR/yamusic-tui`) приводятся к `0700`. Файлы лога, кэша MP3, обложки и metadata — к `0600`, в том числе если они уже существовали с `0755` ([#10](https://github.com/lumpov/yamusic-tui/pull/10)).
- При старте предыдущий `$TMPDIR/yamusic-tui/yamusic-tui.log` удаляется, создаётся новый файл ([#10](https://github.com/lumpov/yamusic-tui/pull/10)).

### Изменено

- Обложка трека не копится в `$TMPDIR/yamusic-tui/{id}.jpg`: каждый раз качается в один `cover.jpg`, файл живёт только пока играет текущий трек (MPRIS `file://`), при следующем треке и при выходе удаляются все `.jpg` в этом каталоге ([#12](https://github.com/lumpov/yamusic-tui/pull/12)).
- Перемотка по умолчанию: `shift+right` / `shift+left` вместо `ctrl+right` / `ctrl+left`. На macOS Control+←/→ забирает Mission Control (переключение рабочих столов), нажатие не доходит до TUI. Alt+стрелки забирают смену панелей в Windows Terminal и перемещение фокуса в оконных менеджерах, где Alt — основная модификаторная клавиша. Shift+стрелки на macOS, Linux и Windows доходят до приложения ([#7](https://github.com/lumpov/yamusic-tui/pull/7)).

### Исправлено

- Текст с метками: текущая строка — последняя, у которой время начала ≤ позиция воспроизведения. После последней метки строка остаётся. Доли секунды LRC `.5` / `.50` / `.500` — 500 мс. Строки `[ar:]`, `[offset:]` и прочие нечисловые теги не становятся текстом ([#30](https://github.com/lumpov/yamusic-tui/pull/30)).
- Позиция текста, полосы и `mm:ss` берётся с часов вывода (`AudioQueueGetCurrentTime` на Darwin, сэмплы oto минус буфер на Linux/Windows), не из доли прочитанных байт стрима ([#30](https://github.com/lumpov/yamusic-tui/pull/30)).
- Громкость и показ текста песни больше не пишут `config.yaml` из `Update`. `config.Save` идёт в `tea.Cmd`, параллельные записи в файл сериализуются ([#29](https://github.com/lumpov/yamusic-tui/pull/29)).
- Клавиша кэша больше не вызывает `BufferAll` из `Update`: докачка хвоста HTTP и запись файла идут в `tea.Cmd` ([#28](https://github.com/lumpov/yamusic-tui/pull/28)).
- Сохранение и удаление кэша трека больше не делают диск из `Update`. Запись и `cache.Remove` идут в `tea.Cmd`. `BufferedStream.WriteTo` копирует снимок буфера и отпускает `mux` до записи на диск ([#27](https://github.com/lumpov/yamusic-tui/pull/27)).
- Лайк и операции с плейлистами (создать, добавить трек, переименовать, удалить) больше не вызывают HTTP из `Update`. Сеть идёт в `tea.Cmd`, `Update` применяет готовый результат ([#26](https://github.com/lumpov/yamusic-tui/pull/26)).
- Darwin Touch Bar / Now Playing: `OnPlayPause` ставит `PlaybackRate` 1 или 0, `OnSeek` — `ElapsedPlaybackTime`, не затирая название трека. `Update` сообщает состояние после Pause/Play ([#25](https://github.com/lumpov/yamusic-tui/pull/25)).
- Windows SMTC: `OnEnded` ставит `MediaPlaybackStatusStopped` ([#25](https://github.com/lumpov/yamusic-tui/pull/25)).
- Чтение закрытого HTTP-тела после полного буфера — `io.EOF`, без `mp3 decoding error` ([#25](https://github.com/lumpov/yamusic-tui/pull/25)).
- `playbacktrace.Enabled` выключен: события не пишутся в лог ([#25](https://github.com/lumpov/yamusic-tui/pull/25)).
- `GET_*` медиакнопок читают tracker в `Update`, а не из горутины `mediaHandle`. `OnPlayback` получает метаданные аргументом и не ждёт `ansChan` на потоке UI ([#25](https://github.com/lumpov/yamusic-tui/pull/25)).
- Загрузка трека, поиск и догрузка ротора больше не вызывают HTTP из `Update`. Сеть идёт в `tea.Cmd`, `Update` применяет готовый результат ([#24](https://github.com/lumpov/yamusic-tui/pull/24)).
- Darwin: колбэк `packetsProc` не ждёт свободный буфер AudioQueue. Pause останавливал колбэки, `waitFreeBuffer` не возвращался, перемотка и play/pause переставали слушаться. Пакеты копятся в списке, enqueue вне колбэка ([#23](https://github.com/lumpov/yamusic-tui/pull/23)).
- `io.EOF` после полного буфера не пишется в `lastError`: Stop не показывает `track buffering` ([#23](https://github.com/lumpov/yamusic-tui/pull/23)).
- TEXT без меток времени снова идёт по прогрессу трека, без `No lyrics timestamps for this track` ([#23](https://github.com/lumpov/yamusic-tui/pull/23)).
- `log.Start` оставляет файл лога, если `Chmod` не удался ([#23](https://github.com/lumpov/yamusic-tui/pull/23)).
- `BufferedStream` не держит `mux` на время `Read`/`ReadFull` сети: Stop из `Update` не ждёт HTTP. Копирование в буфер — под замком ([#23](https://github.com/lumpov/yamusic-tui/pull/23)).
- Windows SMTC: токен `PlaybackPositionChangeRequested` снимается `RemovePlaybackPositionChangeRequested`, а не `RemoveButtonPressed` ([#22](https://github.com/lumpov/yamusic-tui/pull/22)).
- Иконка Play в списке треков берёт `tracker.IsPlaying()`: движение по плейлистам и Back больше не включают её на паузе. Pause/Play обновляют иконку сразу ([#21](https://github.com/lumpov/yamusic-tui/pull/21)).
- Без фокуса окна `View` не пересобирает списки: отдаёт снимок `idleView`. Bubble Tea после каждого `Update` всегда рисует кадр; `skipUIProgress` закрывал только прогресс, а `volumeFadeTickMsg` и прочие сообщения всё равно собирали lipgloss. `volumeFader` шлёт тик только пока громкость не на цели ([#20](https://github.com/lumpov/yamusic-tui/pull/20)).
- Перемотка выравнивает байтовый offset вниз на границу PCM-сэмпла (4 байта, s16le stereo). Было `offset += offset % 4` (1→2, 3→6, не кратно 4) ([#17](https://github.com/lumpov/yamusic-tui/pull/17)).
- Если MP3-декодер не создаётся, oto не получает сырой MP3 (ожидает PCM s16le). Open не вызывается, в трекере `mp3 decoder` ([#16](https://github.com/lumpov/yamusic-tui/pull/16)).
- Если у текста песни нет меток времени, вместо статичных строк показывается `No lyrics timestamps for this track` (как `No lyrics for this track`, когда текста нет) ([#11](https://github.com/lumpov/yamusic-tui/pull/11)).
- После выхода обработчик медиа не закрывает `msgChan`, пока горутины и колбэки ещё могут писать: Windows ждёт `updateTimeline`, macOS обнуляет handler до `close`, Linux MPRIS ставит `closed` после `Stop` ([#15](https://github.com/lumpov/yamusic-tui/pull/15)).
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
