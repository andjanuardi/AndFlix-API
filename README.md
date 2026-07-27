# ANDFLIX-API

Simplified REST API untuk streaming.

Proxy yang menyederhanakan akses ke layanan streaming menjadi endpoint RESTful yang mudah digunakan.

## Requirements

- Go 1.21+

## Build & Run

### Lokal (Windows/Linux)

```bash
go build -o andflix-api .
./andflix-api [port]    # default 9996
```

### GitHub Actions

Push ke branch `main` akan otomatis build binary Linux dan upload sebagai artifact (`andflix-api-linux`).

Push ke branch `main` akan otomatis build binary Linux dan upload sebagai artifact (`andflix-api-linux`).

## Endpoints

### Utama

| Method | Path         | Request                   | Response                               |
| ------ | ------------ | ------------------------- | -------------------------------------- |
| POST   | `/getHome`   | `{page, navId}`           | `{navList, sectionList}`               |
| POST   | `/search`    | `{size, keyword}`         | `{result}`                             |
| POST   | `/getDetail` | `{id, category, episode}` | `{title, playerInfo, recommendations}` |

### Proxy

| Method | Path        | Parameter  | Hasil                    |
| ------ | ----------- | ---------- | ------------------------ |
| GET    | `/image`    | `?url=...` | Gambar (cache 24 jam)    |
| GET    | `/subtitle` | `?url=...` | File SRT (cache 24 jam)  |
| GET    | `/stream`   | `?url=...` | M3U8 / .ts (URL rewrite) |

## Header

Header opsional untuk manajemen session upstream:

| Header              | Keterangan                                          |
| ------------------- | --------------------------------------------------- |
| `X-Device-Id`       | Device ID (dibuat otomatis jika tidak dikirim)       |
| `X-Aeskey-Internal` | AES key internal (dibuat otomatis)                  |
| `X-Token`           | Token upstream (opsional)                            |

API dapat diakses tanpa autentikasi (CORS aktif, semua origin diizinkan).

## Contoh

### getHome

```bash
curl -X POST http://localhost:9996/getHome \
  -H "Content-Type: application/json" \
  -d '{"page": 0, "navId": 1}'
```

### search

```bash
curl -X POST http://localhost:9996/search \
  -H "Content-Type: application/json" \
  -d '{"size": 10, "keyword": "one piece"}'
```

### getDetail

```bash
curl -X POST http://localhost:9996/getDetail \
  -H "Content-Type: application/json" \
  -d '{"id": 103934, "category": 1, "episode": 1}'
```

## Catatan

- Subtitle hanya bahasa Indonesia (`in_ID`, `in`) dan English (`en`, `en_US`)
- URL gambar, subtitle, dan stream otomatis di-rewrite ke proxy endpoint
- `resourceStatus`: 1 → `"Sedang Tayang"`, 2 → `"Selesai"`
- Dokumentasi API lengkap: `GET /docs` (Swagger UI)
- CORS aktif (allow all origins)
