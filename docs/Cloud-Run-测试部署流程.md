# Cloud Run 测试部署流程

状态：CURRENT DEPLOYMENT RUNBOOK
更新时间：2026-05-21

本文记录当前仓库后端以测试模式部署到 Google Cloud Run 的可重复流程。当前目标不是生产安全方案，而是：

- 公开 HTTPS endpoint 可由可信测试前端直接访问。
- `DEV_MODE=true`，后端允许从 `Authorization: Bearer <JWT>` 的 payload 解出 `sub` 作为测试用户。
- 数据库使用 Supabase PostgreSQL，通过 `DATABASE_URL` 连接。
- 媒体 URL 使用 `PUBLIC_ASSET_BASE_URL` 组装。

生产环境不得直接沿用本文的 `DEV_MODE=true` 信任边界。生产应由 API Gateway / Auth provider 校验 JWT 后注入可信 principal，或由后端自行完整校验 Supabase JWT。

## 1. 当前已验证环境

本轮已验证的 GCP 项目：

```text
project_id: project-84868034-4a49-4556-b47
project_name: My First Project
region: us-central1
service_name: lvrs-api
service_url: https://lvrs-api-49376215414.us-central1.run.app
```

已启用 API：

```text
artifactregistry.googleapis.com
cloudbuild.googleapis.com
run.googleapis.com
secretmanager.googleapis.com
```

已创建 Secret Manager secret：

```text
lvrs-database-url
```

当前 Cloud Run 环境变量：

```text
API_ADDR=<from .env>
DEV_MODE=<from .env>
API_GATEWAY_USERINFO_HEADER=<from .env>
PUBLIC_ASSET_BASE_URL=<from .env>
DATABASE_URL=lvrs-database-url:latest
```

## 2. 为什么使用 Dockerfile

最初尝试过：

```bash
gcloud run deploy lvrs-api --source .
```

但本轮实际部署中，Cloud Run source deploy / buildpack 构建失败，且 Cloud Build 日志没有提供足够的 buildpack stderr。为了得到可重复、可排障的部署路径，当前采用：

1. `Dockerfile` 多阶段构建 Go server。
2. `gcloud builds submit --tag ...` 构建并推送镜像到 Artifact Registry。
3. `gcloud run deploy --image ...` 部署镜像。

这条路径已经验证成功。

## 3. 仓库部署文件

当前部署依赖以下文件：

```text
Dockerfile
.dockerignore
.gcloudignore
```

`Dockerfile` 构建入口是：

```text
./cmd/server
```

`.gcloudignore` 必须避免从 `.gitignore` 继承 `server` 规则后误排除 `cmd/server`。如果忽略文件配置错误，Cloud Build 可能报：

```text
stat /src/cmd/server: directory not found
```

因此忽略根目录构建产物时应写成：

```text
/server
```

不要写成：

```text
server
```

## 4. 一次性项目准备

切换项目：

```bash
gcloud config set project project-84868034-4a49-4556-b47
gcloud config set run/region us-central1
```

启用 API：

```bash
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com
```

确认 billing：

```bash
gcloud billing projects describe project-84868034-4a49-4556-b47 \
  --format='yaml(billingEnabled,billingAccountName)'
```

## 5. 本地 `.env` 准备

部署命令严格从本地 `.env` 读取运行时配置。当前 `cmd/server` 读取以下字段：

```text
DATABASE_URL
PUBLIC_ASSET_BASE_URL
API_ADDR
DEV_MODE
API_GATEWAY_USERINFO_HEADER
```

测试部署建议 `.env` 显式包含：

```dotenv
DATABASE_URL=postgresql://...
PUBLIC_ASSET_BASE_URL=https://storage.googleapis.com/videos2077
API_ADDR=:8080
DEV_MODE=true
API_GATEWAY_USERINFO_HEADER=X-Apigateway-Api-Userinfo
```

说明：

- `DATABASE_URL` 只用于创建 / 更新 Secret Manager，不直接写入 Cloud Run 明文 env。
- 其他字段会从 dotenv 格式的 `.env` 派生成临时 YAML 文件，再通过 `--env-vars-file` 写入 Cloud Run revision。`gcloud run deploy --env-vars-file` 不接受 `KEY=value` dotenv 文件。
- 如果 `.env` 没有 `DEV_MODE=true`，部署出来的服务就不应是 DEV_MODE。

部署前可用下面命令只检查 key 是否存在：

```bash
for key in DATABASE_URL PUBLIC_ASSET_BASE_URL API_ADDR DEV_MODE API_GATEWAY_USERINFO_HEADER; do
  awk -F= -v key="$key" '$1==key && substr($0,index($0,"=")+1)!="" { found=1 } END { exit found ? 0 : 1 }' .env \
    || { echo "$key missing in .env"; exit 1; }
done
```

## 6. Secret 准备

不要把 `DATABASE_URL` 明文写进部署命令。用本地 `.env` 创建或更新 Secret Manager：

```bash
DATABASE_URL="$(awk -F= '$1=="DATABASE_URL"{print substr($0,index($0,"=")+1); exit}' .env)"

if gcloud secrets describe lvrs-database-url >/dev/null 2>&1; then
  printf '%s' "$DATABASE_URL" | \
    gcloud secrets versions add lvrs-database-url --data-file=-
else
  printf '%s' "$DATABASE_URL" | \
    gcloud secrets create lvrs-database-url \
      --replication-policy=automatic \
      --data-file=-
fi
```

给 Cloud Run 默认运行服务账号读取 secret 的权限：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
RUN_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

gcloud secrets add-iam-policy-binding lvrs-database-url \
  --member="serviceAccount:${RUN_SERVICE_ACCOUNT}" \
  --role='roles/secretmanager.secretAccessor'
```

## 7. Cloud Build 权限

本项目当前 Cloud Build 使用默认 Compute service account 执行构建。为了能读取源码包、写日志、推送 Artifact Registry，需要以下权限：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
BUILD_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${BUILD_SERVICE_ACCOUNT}" \
  --role='roles/storage.objectViewer'

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${BUILD_SERVICE_ACCOUNT}" \
  --role='roles/artifactregistry.writer'

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${BUILD_SERVICE_ACCOUNT}" \
  --role='roles/logging.logWriter'
```

如果缺少这些权限，常见错误包括：

```text
storage.objects.get denied
artifactregistry.repositories.uploadArtifacts denied
does not have permission to write logs to Cloud Logging
```

## 8. 数据库迁移检查

部署前确认 Supabase schema 已经是最新：

```bash
make analytics-migrate-status
make catalog-migrate-status
make learningengine-migrate-status
make recommendation-migrate-status
```

本轮部署前已验证：

```text
analytics: current=5 applied=5 pending=0
catalog: current=12 applied=12 pending=0
learningengine: current=6 applied=6 pending=0
recommendation: current=7 applied=7 pending=0
```

如有 pending，先执行：

```bash
make analytics-migrate-up
make catalog-migrate-up
make learningengine-migrate-up
make recommendation-migrate-up
```

## 9. 构建镜像

从仓库根目录执行：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
IMAGE="us-central1-docker.pkg.dev/${PROJECT_ID}/cloud-run-source-deploy/lvrs-api:manual-$(date +%Y%m%d%H%M%S)"
echo "$IMAGE" > /tmp/lvrs-api-image.txt

gcloud builds submit \
  --region us-central1 \
  --tag "$IMAGE" \
  .
```

成功时会看到：

```text
STATUS
SUCCESS
```

并且镜像会被推送到：

```text
us-central1-docker.pkg.dev/<project_id>/cloud-run-source-deploy/lvrs-api:<tag>
```

## 10. 部署 Cloud Run

使用刚构建好的镜像部署：

```bash
IMAGE="$(cat /tmp/lvrs-api-image.txt)"

runtime_env_file="$(mktemp)"
awk -F= '
  $1=="PUBLIC_ASSET_BASE_URL" || $1=="API_ADDR" || $1=="DEV_MODE" || $1=="API_GATEWAY_USERINFO_HEADER" {
    key=$1
    value=substr($0,index($0,"=")+1)
    gsub(/\\/, "\\\\", value)
    gsub(/"/, "\\\"", value)
    printf "%s: \"%s\"\n", key, value
  }
' .env > "$runtime_env_file"

gcloud run deploy lvrs-api \
  --image "$IMAGE" \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --env-vars-file "$runtime_env_file" \
  --set-secrets DATABASE_URL=lvrs-database-url:latest

rm -f "$runtime_env_file"
```

成功输出类似：

```text
Service [lvrs-api] revision [lvrs-api-00001-tm8] has been deployed and is serving 100 percent of traffic.
Service URL: https://lvrs-api-49376215414.us-central1.run.app
```

同一个 service name `lvrs-api` 重新部署时会创建新 revision，但 service URL 保持稳定。

## 11. 验证

查看服务状态：

```bash
gcloud run services describe lvrs-api \
  --region us-central1 \
  --format='yaml(status.url,status.conditions,status.latestReadyRevisionName,status.traffic)'
```

确认环境变量：

```bash
gcloud run services describe lvrs-api \
  --region us-central1 \
  --format='yaml(spec.template.spec.containers[0].env,status.latestReadyRevisionName,status.url)'
```

公网 HTTPS 可达性：

```bash
curl -i https://lvrs-api-49376215414.us-central1.run.app/
```

当前根路径没有 handler，返回 `404 page not found` 是预期；它只证明 HTTPS endpoint 和 Cloud Run 路由可达。

未带测试 token 的认证验证：

```bash
curl -i https://lvrs-api-49376215414.us-central1.run.app/api/me
```

预期：

```text
401 unauthorized
```

带 DEV_MODE 测试 token：

```bash
TOKEN="$(printf '{"alg":"none"}' | base64 | tr '+/' '-_' | tr -d '=')"."$(printf '{"sub":"test-user-001"}' | base64 | tr '+/' '-_' | tr -d '=')".

curl -i \
  -H "Authorization: Bearer ${TOKEN}" \
  https://lvrs-api-49376215414.us-central1.run.app/api/me
```

说明：

- 如果返回 `401`，说明 token payload 没有被 DEV_MODE auth 接受。
- 如果返回 `500`，说明请求已经进入服务并解析出 `sub`，但业务层或数据库数据不满足该 endpoint 当前依赖。
- 当前本轮验证中，`/api/me` 带 `sub=test-user-001` 返回过 `500`，日志显示 `user_id=test-user-001`，所以 Cloud Run 部署和 DEV_MODE auth 已经生效。

查看日志：

```bash
gcloud run services logs read lvrs-api \
  --region us-central1 \
  --limit 100
```

更细的结构化日志：

```bash
gcloud logging read \
  'resource.type="cloud_run_revision" AND resource.labels.service_name="lvrs-api"' \
  --freshness=10m \
  --limit=100 \
  --format=json
```

## 12. 重复部署最短命令

如果 API、IAM、secret、migration 都已经准备好，之后重复部署只需要：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
IMAGE="us-central1-docker.pkg.dev/${PROJECT_ID}/cloud-run-source-deploy/lvrs-api:manual-$(date +%Y%m%d%H%M%S)"
runtime_env_file="$(mktemp)"
awk -F= '
  $1=="PUBLIC_ASSET_BASE_URL" || $1=="API_ADDR" || $1=="DEV_MODE" || $1=="API_GATEWAY_USERINFO_HEADER" {
    key=$1
    value=substr($0,index($0,"=")+1)
    gsub(/\\/, "\\\\", value)
    gsub(/"/, "\\\"", value)
    printf "%s: \"%s\"\n", key, value
  }
' .env > "$runtime_env_file"

gcloud builds submit \
  --region us-central1 \
  --tag "$IMAGE" \
  .

gcloud run deploy lvrs-api \
  --image "$IMAGE" \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --env-vars-file "$runtime_env_file" \
  --set-secrets DATABASE_URL=lvrs-database-url:latest

rm -f "$runtime_env_file"
```

## 13. 生产化前必须调整

测试部署当前刻意使用：

```text
--allow-unauthenticated
DEV_MODE=true
```

生产化前至少需要处理：

- 关闭 `DEV_MODE`。
- 不再信任客户端可伪造的 header 或未验签 JWT payload。
- 接入 API Gateway / Auth provider，或后端完整校验 Supabase JWT。
- 增加 `/healthz` endpoint，便于健康检查和排障。
- 明确 Cloud Run service account 最小权限，而不是继续复用默认 Compute service account。
- 为 GCS 视频访问确定公开 bucket、signed URL 或 CDN 策略。
