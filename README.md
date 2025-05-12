# 📸 Media Import CLI Tool

A flexible, template-driven CLI tool written in Go to import and organize photo and video files from a base directory to a structured output directory.

---

## 🚀 Features

- 🔍 Recursive scanning of input directories
- 🏷️ Organizes files using a text/template path format
- 📦 Parallel file copying with configurable workers
- 🧪 Dry-run mode (preview actions without copying)
- 🎚️ Real-time progress bars per file
- 🎛️ Configurable chunk size (e.g. `32k`, `1m`)
- 🎯 Customizable allowed file extensions
- 📈 Displays overall transfer rate

---

## 🛠️ Installation

Install from URL:
```bash
go install github.com/yourusername/media_import@latest
```

Install from a git clone:
```bash
git clone https://github.com/markuscraig/media_import.git
cd media_import
```

Build locally:
```bash
go build -o media_import main.go
```

---

## 📝 Usage

```bash
media_import --help

Usage:
  -chunk-size string
    	Copy buffer size (e.g. 32k, 1m) (default "1m")
  -dry-run
    	Log actions without copying files
  -extensions string
    	Comma-separated list of file extensions to import (default ".jpg,.jpeg,.png,.gif,.bmp,.mp4,.mov,.avi,.mkv")
  -input string
    	Input directory root (required)
  -output string
    	Output directory root (required)
  -overwrite
    	Overwrite existing output files
  -template string
    	Text template for organizing output paths (default "{{.Year}}/{{.Year}}-{{.Month}}-{{.Day}}/{{.Name}}")
  -workers int
    	Number of parallel workers (default 3)
```

---

## 🔧 CLI Options

| Flag             | Description |
|------------------|-------------|
| `--input`        | **(Required)** Path to input directory |
| `--output`       | **(Required)** Path to output directory |
| `--template`     | Template path for organizing output (default: `{{.Year}}/{{.Year}}-{{.Month}}-{{.Day}}/{{.Name}}`) |
| `--extensions`   | Comma-separated list of file extensions to import (default: `.jpg,.jpeg,.png,.gif,.bmp,.mp4,.mov,.avi,.mkv`) |
| `--workers`      | Number of parallel file copying workers (default: `4`) |
| `--chunk-size`   | Copy buffer size (supports `k`/`m`, default: `32k`) |
| `--dry-run`      | If enabled, logs operations without copying files |

---

## 🧪 Example Commands

### 🔹 Basic import using defaults

Relative paths are supported:
```bash
media_import --input /Volumes/Photos --output ./Backup
```

---

### 🔹 Organize by year/month/filename

```bash
media_import \
  --input /Volumes/Photos \
  --output ./Backup \
  --template "{{.Year}}/{{.Month}}/{{.Name}}"
```

---

### 🔹 Only import .mp4 and .mov files

```bash
media_import \
  --input /Volumes/GoPro \
  --output ./Videos \
  --extensions ".mp4,.mov"
```

---

### 🔹 Show actions without copying

```bash
media_import \
  --input /Volumes/Camera \
  --output ./DryRunTest \
  --dry-run
```

---

### 🔹 Use 8 workers and 1MB chunk size

```bash
media_import \
  --input ./Media \
  --output ./Archive \
  --workers 8 \
  --chunk-size 1m
```

---

## 📂 Template Variables

Customize the output directory structure using the Go `text/template` syntax:

| Variable | Description             |
|----------|-------------------------|
| `Year`   | File's year (YYYY)      |
| `Month`  | File's month (MM)       |
| `Day`    | File's day (DD)         |
| `Name`   | File name with extension |

---

### ✨ Example Template:

```bash
--template "{{.Year}}/{{.Year}}-{{.Month}}-{{.Day}}/{{.Name}}"
```

---

## 📈 Output Example

- Each file's progress is shown as text progress bars.
- The total average bandwidth is displayed when finished.

```text
Worker 0: beach.jpg [==========>            ] 50%
Worker 1: skate.mp4 [======================>] 100%
Worker 2: snow.png  [===============>       ] 80%
Total copied: 413.50 MB in 6.23 seconds (66.35 MB/s)
```

---

## ⚠️ Important Notes

- Existing files **are not overwritten** (by default).
- Directories will be created if they don't exist.
- Existing directories are **not deleted**.

---

## 📃 License

MIT – Use it, fork it, improve it. Contributions welcome!

---

## 🙌 Contributing

Feel free to submit an issue or PR if you have suggestions or improvements!

