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
project_id: <gcp-project-id>
project_name: <gcp-project-name>
region: <gcp-region>
service_name: <cloud-run-service-name>
service_url: <cloud-run-service-url>
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
<database-url-secret-name>
```

当前 Cloud Run 环境变量：

```text
API_ADDR=<from .env>
DEV_MODE=<from .env>
API_GATEWAY_USERINFO_HEADER=<from .env>
PUBLIC_ASSET_BASE_URL=<from .env>
DATABASE_URL=<database-url-secret-name>:latest
```

## 2. 为什么使用 Dockerfile

最初尝试过：

```bash
gcloud run deploy <cloud-run-service-name> --source .
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
gcloud config set project <gcp-project-id>
gcloud config set run/region <gcp-region>
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
gcloud billing projects describe <gcp-project-id> \
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
PUBLIC_ASSET_BASE_URL=<public-asset-base-url>
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

if gcloud secrets describe <database-url-secret-name> >/dev/null 2>&1; then
  printf '%s' "$DATABASE_URL" | \
    gcloud secrets versions add <database-url-secret-name> --data-file=-
else
  printf '%s' "$DATABASE_URL" | \
    gcloud secrets create <database-url-secret-name> \
      --replication-policy=automatic \
      --data-file=-
fi
```

给 Cloud Run 默认运行服务账号读取 secret 的权限：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
RUN_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

gcloud secrets add-iam-policy-binding <database-url-secret-name> \
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
IMAGE="<gcp-region>-docker.pkg.dev/${PROJECT_ID}/cloud-run-source-deploy/<cloud-run-service-name>:manual-$(date +%Y%m%d%H%M%S)"
echo "$IMAGE" > /tmp/<cloud-run-service-name>-image.txt

gcloud builds submit \
  --region <gcp-region> \
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
<gcp-region>-docker.pkg.dev/<project_id>/cloud-run-source-deploy/<cloud-run-service-name>:<tag>
```

## 10. 部署 Cloud Run

使用刚构建好的镜像部署：

```bash
IMAGE="$(cat /tmp/<cloud-run-service-name>-image.txt)"

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

gcloud run deploy <cloud-run-service-name> \
  --image "$IMAGE" \
  --region <gcp-region> \
  --allow-unauthenticated \
  --port 8080 \
  --env-vars-file "$runtime_env_file" \
  --set-secrets DATABASE_URL=<database-url-secret-name>:latest

rm -f "$runtime_env_file"
```

成功输出类似：

```text
Service [<cloud-run-service-name>] revision [<cloud-run-service-name>-00001-tm8] has been deployed and is serving 100 percent of traffic.
Service URL: <cloud-run-service-url>
```

同一个 service name `<cloud-run-service-name>` 重新部署时会创建新 revision，但 service URL 保持稳定。

## 11. 验证

查看服务状态：

```bash
gcloud run services describe <cloud-run-service-name> \
  --region <gcp-region> \
  --format='yaml(status.url,status.conditions,status.latestReadyRevisionName,status.traffic)'
```

确认环境变量：

```bash
gcloud run services describe <cloud-run-service-name> \
  --region <gcp-region> \
  --format='yaml(spec.template.spec.containers[0].env,status.latestReadyRevisionName,status.url)'
```

公网 HTTPS 可达性：

```bash
curl -i <cloud-run-service-url>/
```

当前根路径没有 handler，返回 `404 page not found` 是预期；它只证明 HTTPS endpoint 和 Cloud Run 路由可达。

未带测试 token 的认证验证：

```bash
curl -i <cloud-run-service-url>/api/me
```

预期：

```text
401 unauthorized
```

带 DEV_MODE 测试 token：

```bash
TOKEN="$(printf '{"alg":"none"}' | base64 | tr '+/' '-_' | tr -d '=')"."$(printf '{"sub":"<test-user-id>"}' | base64 | tr '+/' '-_' | tr -d '=')".

curl -i \
  -H "Authorization: Bearer ${TOKEN}" \
  <cloud-run-service-url>/api/me
```

说明：

- 如果返回 `401`，说明 token payload 没有被 DEV_MODE auth 接受。
- 如果返回 `500`，说明请求已经进入服务并解析出 `sub`，但业务层或数据库数据不满足该 endpoint 当前依赖。
- 当前本轮验证中，`/api/me` 带 `sub=<test-user-id>` 返回过 `500`，日志显示 `user_id=<test-user-id>`，所以 Cloud Run 部署和 DEV_MODE auth 已经生效。

查看日志：

```bash
gcloud run services logs read <cloud-run-service-name> \
  --region <gcp-region> \
  --limit 100
```

更细的结构化日志：

```bash
gcloud logging read \
  'resource.type="cloud_run_revision" AND resource.labels.service_name="<cloud-run-service-name>"' \
  --freshness=10m \
  --limit=100 \
  --format=json
```

## 12. 重复部署最短命令

如果 API、IAM、secret、migration 都已经准备好，之后重复部署只需要：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
IMAGE="<gcp-region>-docker.pkg.dev/${PROJECT_ID}/cloud-run-source-deploy/<cloud-run-service-name>:manual-$(date +%Y%m%d%H%M%S)"
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
  --region <gcp-region> \
  --tag "$IMAGE" \
  .

gcloud run deploy <cloud-run-service-name> \
  --image "$IMAGE" \
  --region <gcp-region> \
  --allow-unauthenticated \
  --port 8080 \
  --env-vars-file "$runtime_env_file" \
  --set-secrets DATABASE_URL=<database-url-secret-name>:latest

rm -f "$runtime_env_file"
```

## 13. 绑定 Cloudflare 域名

当前 `<root-domain>` 已经在 Cloudflare 账号 `<cloudflare-account-name>` 中接入，且同一个域名已经承载：

```text
<root-domain>       A      -> <existing-web-host>
www.<root-domain>   CNAME  -> <existing-web-host>
MX / TXT / DKIM         -> <email-provider> 邮箱
```

因此后端 API 不绑定根域名，也不改 `www`，只使用独立子域名：

```text
<api-domain>
```

### 13.1 当前 Cloudflare DNS 保护边界

绑定 API 域名时不要修改以下记录：

```text
<root-domain> A
www.<root-domain> CNAME
<root-domain> MX
<root-domain> TXT
sig1._domainkey.<root-domain> CNAME
```

本轮只新增了：

```text
<api-domain> CNAME ghs.googlehosted.com
```

并且保持：

```text
proxied=false
DNS only / 灰云
```

证书签发完成前不要打开 Cloudflare 代理。Cloudflare 代理可能影响 Google managed certificate 的验证和续期。

### 13.2 GCP 创建 Cloud Run domain mapping

Cloud Run fully managed domain mapping 需要 `gcloud beta`：

```bash
gcloud components install beta --quiet
```

创建 mapping：

```bash
gcloud beta run domain-mappings create \
  --service <cloud-run-service-name> \
  --domain <api-domain> \
  --region <gcp-region>
```

本轮创建成功后，GCP 返回的 DNS 要求是：

```text
NAME  RECORD TYPE  CONTENTS
api   CNAME        ghs.googlehosted.com.
```

查看 mapping 状态：

```bash
gcloud beta run domain-mappings describe \
  --domain <api-domain> \
  --region <gcp-region> \
  --format='yaml(status.resourceRecords,status.conditions,metadata.name)'
```

本轮当前状态：

```text
DomainRoutable: True
CertificateProvisioned: Unknown
Ready: CertificatePending
```

这表示 DNS 已经可路由，但 Google managed certificate 仍在自动签发。证书未完成前，`https://<api-domain>` 可能出现 SSL 连接错误，这是预期状态。

### 13.3 Cloudflare 添加 DNS 记录

在 Cloudflare 的 `<root-domain>` zone 中添加：

```text
Type: CNAME
Name: api
Target: ghs.googlehosted.com
Proxy status: DNS only
TTL: Auto
Comment: Cloud Run <cloud-run-service-name>
```

本轮通过 Cloudflare API 创建的记录为：

```text
type: CNAME
name: <api-domain>
content: ghs.googlehosted.com
proxied: false
ttl: 1
comment: Cloud Run <cloud-run-service-name>
```

DNS 验证：

```bash
dig +short <api-domain> CNAME
dig +short <api-domain> A
```

本轮已验证：

```text
<api-domain> CNAME -> ghs.googlehosted.com.
```

### 13.4 等待证书完成

重复检查：

```bash
gcloud beta run domain-mappings describe \
  --domain <api-domain> \
  --region <gcp-region> \
  --format='yaml(status.conditions)'
```

当看到：

```text
Ready: True
CertificateProvisioned: True
```

再验证 HTTPS：

```bash
curl -i https://<api-domain>/api/me
```

未带 token 时预期返回：

```text
401 unauthorized
```

这表示完整链路成功：

```text
<api-domain>
  -> Cloudflare DNS only
  -> ghs.googlehosted.com
  -> Cloud Run domain mapping
  -> <cloud-run-service-name>
```

### 13.5 以后是否打开 Cloudflare 代理

测试阶段建议继续保持 DNS only。

如果之后要打开 Cloudflare 橙云代理，先确认 Cloud Run domain mapping 已经 `Ready=True`，然后：

- Cloudflare DNS record 改为 proxied。
- Cloudflare SSL/TLS mode 使用 `Full (strict)`。
- 避免启用会干扰 Google certificate renewal 的强制跳转规则，尤其是证书验证路径相关规则。

## 14. 生产化前必须调整

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
