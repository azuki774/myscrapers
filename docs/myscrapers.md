# myscrapers (Go) — Usage Guide

myscrapers is a containerised Go CLI that scrapes MoneyForward household-accounting pages and either writes CSV files locally or uploads them to S3. It replaces the legacy Python scraper under `src/moneyforward/`.

## Binary

```
myscraper moneyforward [flags]
```

The Docker image sets `ENTRYPOINT ["myscraper"]` and `CMD ["moneyforward", "--fetch", "--s3-upload"]`, so running the container without arguments executes a fetch with S3 upload.

## Commands

Exactly **one** of `--fetch` or `--update` is required.

| Flag | Description |
|------|-------------|
| `--fetch` | Scrape the current-month CF table, last-month CF table, and asset-history page. Write three CSV files (`cf.csv`, `cf_lastmonth.csv`, `asset_history.csv`) and convert them to Shift-JIS. |
| `--update` | Press the "一括更新" (bulk update) button on the accounts page, then click the モバイルSuica "更新" link on the home page. |

## Flags (args)

| Flag | Default | Env fallback | Description |
|------|---------|--------------|-------------|
| `--fetch` | `false` | — | Run the fetch subcommand |
| `--update` | `false` | — | Run the update subcommand |
| `--s3-upload` | `false` | — | (with `--fetch` only) Upload the three CSVs to S3 after writing. Ignored with a warning when combined with `--update`. |
| `--cookie-path` | `/data/cookie.json` | `MF_COOKIE_PATH` | Path to the browser-exported cookie JSON file |
| `--output-dir` | `/data` | `MF_OUTPUT_DIR` | Directory where CSV files are written |
| `--headless` | `true` | — | Run the browser in headless mode (currently always true in production) |

### `--s3-upload` in detail

When `--s3-upload` is passed together with `--fetch`, the binary builds an `S3Uploader` after writing the CSVs and uploads each file to S3.

**Required environment variables:**

| Variable | Description |
|----------|-------------|
| `BUCKET_URL` | S3-compatible endpoint URL (e.g. `https://s3.ap-northeast-1.amazonaws.com`) |
| `BUCKET_NAME` | Target bucket name |
| `BUCKET_DIR` | Object-key prefix (e.g. `myscrapers/moneyforward`) |
| `AWS_REGION` | AWS region (e.g. `ap-northeast-1`) |
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |

**Upload destinations** (prefix = `BUCKET_DIR`):

```
s3://<BUCKET_NAME>/<BUCKET_DIR>/cf.csv
s3://<BUCKET_NAME>/<BUCKET_DIR>/cf_lastmonth.csv
s3://<BUCKET_NAME>/<BUCKET_DIR>/asset_history.csv
```

The uploader uses path-style addressing (`UsePathStyle = true`), which is compatible with AWS S3 and most S3-compatible services (MinIO, R2, etc.).

## Volume

The container declares a single volume at `/data`:

```dockerfile
VOLUME ["/data"]
```

| Path inside container | Purpose |
|-----------------------|---------|
| `/data/cookie.json` | Cookie file read at startup (override with `--cookie-path` or `MF_COOKIE_PATH`) |
| `/data/cf.csv` | Current-month cash-flow CSV (output) |
| `/data/cf_lastmonth.csv` | Last-month cash-flow CSV (output) |
| `/data/asset_history.csv` | Asset history CSV (output) |
| `/data/*.html` | Debug HTML files written on parse errors |

In production, mount a PersistentVolume or hostPath at `/data` so cookies persist across runs and CSVs are accessible after the container exits.

## Environment variables summary

| Variable | Used by | Description |
|----------|---------|-------------|
| `MF_COOKIE_PATH` | CLI | Cookie file path (default `/data/cookie.json`) |
| `MF_OUTPUT_DIR` | CLI | Output directory (default `/data`) |
| `TZ` | Runtime | Timezone; set to `Asia/Tokyo` in the image |
| `PLAYWRIGHT_DRIVER_PATH` | Playwright | Where to cache the Playwright browser driver |
| `BUCKET_URL` | `--s3-upload` | S3 endpoint URL |
| `BUCKET_NAME` | `--s3-upload` | S3 bucket name |
| `BUCKET_DIR` | `--s3-upload` | S3 object-key prefix |
| `AWS_REGION` | `--s3-upload` | AWS region |
| `AWS_ACCESS_KEY_ID` | `--s3-upload` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | `--s3-upload` | AWS secret key |

## Kubernetes workload

myscrapers is designed to run as a **Kubernetes CronJob** that executes on a schedule (e.g. daily). Below is a reference manifest.

### CronJob example

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: myscrapers-fetch
  namespace: myscrapers
spec:
  schedule: "0 3 * * *"            # every day at 03:00
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: myscrapers
              image: ghcr.io/azuki774/myscrapers:latest
              args: ["moneyforward", "--fetch", "--s3-upload"]
              env:
                - name: TZ
                  value: Asia/Tokyo
                - name: MF_OUTPUT_DIR
                  value: /data
                - name: MF_COOKIE_PATH
                  value: /data/cookie.json
                - name: BUCKET_URL
                  valueFrom:
                    secretKeyRef:
                      name: myscrapers-s3
                      key: bucket-url
                - name: BUCKET_NAME
                  valueFrom:
                    secretKeyRef:
                      name: myscrapers-s3
                      key: bucket-name
                - name: BUCKET_DIR
                  valueFrom:
                    secretKeyRef:
                      name: myscrapers-s3
                      key: bucket-dir
                - name: AWS_REGION
                  valueFrom:
                    secretKeyRef:
                      name: myscrapers-s3
                      key: aws-region
                - name: AWS_ACCESS_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      name: myscrapers-s3
                      key: aws-access-key-id
                - name: AWS_SECRET_ACCESS_KEY
                  valueFrom:
                    secretKeyRef:
                      name: myscrapers-s3
                      key: aws-secret-access-key
              volumeMounts:
                - name: data
                  mountPath: /data
          volumes:
            - name: data
              persistentVolumeClaim:
                claimName: myscrapers-data
```

### Separate CronJob for `--update`

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: myscrapers-update
  namespace: myscrapers
spec:
  schedule: "0 6 * * *"            # every day at 06:00
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      backoffLimit: 1
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: myscrapers
              image: ghcr.io/azuki774/myscrapers:latest
              args: ["moneyforward", "--update"]
              env:
                - name: TZ
                  value: Asia/Tokyo
                - name: MF_COOKIE_PATH
                  value: /data/cookie.json
              volumeMounts:
                - name: data
                  mountPath: /data
          volumes:
            - name: data
              persistentVolumeClaim:
                claimName: myscrapers-data
```

### Key points for Kubernetes

- **`args`** overrides the image `CMD`. Use `args: ["moneyforward", "--fetch", "--s3-upload"]` for fetch or `args: ["moneyforward", "--update"]` for update.
- **Volume**: Mount a PersistentVolumeClaim at `/data` so the cookie file persists between CronJob runs and CSVs survive until the next run.
- **Secrets**: Store `BUCKET_*` and `AWS_*` values in a Kubernetes Secret and inject them via `secretKeyRef`. Never put credentials in the image or manifest directly.
- **`concurrencyPolicy: Forbid`** prevents overlapping runs if a scrape takes longer than the schedule interval.
- **Cookie management**: The cookie file at `/data/cookie.json` must be refreshed periodically (exported from a browser session). Consider an init container or a separate job that updates it.

## Quick reference — common invocations

```bash
# Docker: fetch with S3 upload (image default CMD)
docker run --rm \
  -v ./data:/data \
  -e BUCKET_URL=https://s3.example.com \
  -e BUCKET_NAME=my-bucket \
  -e BUCKET_DIR=myscrapers/moneyforward \
  -e AWS_REGION=ap-northeast-1 \
  -e AWS_ACCESS_KEY_ID=xxx \
  -e AWS_SECRET_ACCESS_KEY=xxx \
  myscrapers

# Docker: fetch without S3 upload
docker run --rm -v ./data:/data myscrapers \
  myscraper moneyforward --fetch

# Docker: update (press bulk-update + Suica)
docker run --rm -v ./data:/data myscrapers \
  myscraper moneyforward --update

# Kubernetes: see CronJob manifests above
```
