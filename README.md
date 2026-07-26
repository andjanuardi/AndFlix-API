# ANDFLIX-API

Simplified REST API untuk streaming.

## Requirements

- Go 1.21+

## Run

```bash
go build -o andflix-api.exe .
andflix-api.exe [port]    # default 8080
```

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

## Session

Header untuk persist session:

| Header              | Keterangan                                                          |
| ------------------- | ------------------------------------------------------------------- |
| `X-Device-Id`       | Device identity. Simpan dari response, kirim di request berikutnya. |
| `X-Aeskey-Internal` | AES key. Simpan dari response, kirim di request berikutnya.         |
| `X-Token`           | JWT untuk akses terautentikasi.                                     |

## Contoh

### getHome

```bash
curl -X POST http://localhost:8080/getHome \
  -H "Content-Type: application/json" \
  -d '{"page": 0, "navId": 1}'
```

### search

```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{"size": 10, "keyword": "one piece"}'
```

### getDetail

```bash
curl -X POST http://localhost:8080/getDetail \
  -H "Content-Type: application/json" \
  -d '{"id": 103934, "category": 1, "episode": 1}'
```

## Catatan

- Subtitle hanya bahasa Indonesia (`in_ID`, `in`) dan English (`en`, `en_US`)
- URL gambar, subtitle, dan stream otomatis di-rewrite ke proxy endpoint
- `resourceStatus`: 1 → `"Sedang Tayang"`, 2 → `"Selesai"`
- Build: `go vet` clean
