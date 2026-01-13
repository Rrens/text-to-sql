# 🚀 Panduan Build & Push ke Docker Hub (Multi-Platform)

Panduan ini berisi langkah-langkah untuk meng-update image `rrenss/text-2-sql` di Docker Hub agar bisa jalan di semua OS (Linux, Mac, Windows).

## 1. Persiapan Awal (Hanya Sekali)

Sebelum mulai, pastikan kamu sudah login dan menyiapkan builder agar bisa bikin image untuk Linux (AMD64) dan Mac (ARM64) sekaligus.

### A. Login ke Docker Hub

Buka terminal dan jalankan:

```bash
docker login
```

_Masukkan username (`rrenss`) dan password kamu jika diminta._

### B. Siapkan Builder (Buildx)

Docker standar kadang tidak bisa build multi-platform secara default. Buat builder baru dengan perintah ini:

```bash
docker buildx create --use --name multiarch-builder --driver docker-container
docker buildx inspect --bootstrap
```

_(Kalau sudah pernah dibuat, perintah ini mungkin error "already exists", abaikan saja)._

---

## 2. Cara Update / Push Image Baru

Setiap kali kamu ada perubahan code dan ingin update di Docker Hub, jalankan perintah "Sakti" ini.

Perintah ini akan:

1.  Build Backend (Go) & Frontend (React).
2.  Menggabungkannya jadi 1 image.
3.  Membuat versi untuk **Linux (AMD64)** dan **Mac (ARM64)**.
4.  Otomatis **Push** ke Docker Hub.

**Jalankan di root folder project:**

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t rrenss/text-2-sql:latest \
  -f deployments/docker/Dockerfile.unified \
  . \
  --push
```

_Proses ini butuh waktu beberapa menit (tergantung koneksi internet) karena dia harus compile 2x (untuk 2 arsitektur)._

---

## 3. Verifikasi

Setelah selesai, kamu bisa cek di website Docker Hub, atau coba pull ulang:

```bash
docker pull rrenss/text-2-sql:latest
docker run --rm rrenss/text-2-sql:latest --version
```

---

## 4. Cara Deploy di Server (Production)

> **PENTING:** Jangan gunakan `deployments/docker/docker-compose.yaml` di server, karena file itu untuk development dan membutuhkan full source code.

Gunakan paket distribusi `text-to-sql-dist.tar.gz` yang sudah disiapkan. Paket ini menggunakan image yang sudah jadi (`rrenss/text-2-sql:latest`).

### Langkah-langkah:

### Langkah-langkah:

1.  **Dapatkan File Config:**

    - **Jika Repo Public:**
      ```bash
      curl -sL https://raw.githubusercontent.com/Rrens/text-to-sql/main/deploy-pkg/docker-compose.yaml > docker-compose.yaml
      ```
    - **Jika Repo Private (Gunakan Secret Gist):**
      1. Buka [gist.github.com](https://gist.github.com).
      2. Buat Gist baru (pilih "Create secret gist").
      3. Copas isi file `deploy-pkg/docker-compose.yaml` ke sana.
      4. Klik tombol "Raw" di Gist yang sudah jadi, copy URL-nya.
      5. Gunakan URL itu:
         ```bash
         curl -sL https://gist.githubusercontent.com/USER/ID/raw/docker-compose.yaml > docker-compose.yaml
         ```

2.  **Jalankan Aplikasi:**
    ```bash
    docker compose up -d
    ```

## Aplikasi akan berjalan di background. Port default API adalah **4081**.

## Catatan Penting

- **Kenapa harus `--platform`?**
  Supaya temanmu yang pakai Linux Server (VPS) atau Windows laptop biasa (AMD/Intel) bisa pakai, gak cuma yang pake Macbook M1/M2/M3 aja.
- **Error connection/timeout?**
  Coba restart Docker Desktop kamu. Membangun multi-platform butuh resource lumayan.
