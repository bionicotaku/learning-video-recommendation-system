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
compute.googleapis.com
certificatemanager.googleapis.com
networkservices.googleapis.com
run.googleapis.com
secretmanager.googleapis.com
```

已创建 Secret Manager secret：

```text
<database-url-secret-name>
```

当前 Cloud Run 环境变量：

```text
DATABASE_URL=<database-url-secret-name>:latest
PUBLIC_ASSET_BASE_URL=<public-asset-base-url-secret-name>:latest
API_ADDR=<api-addr-secret-name>:latest
DEV_MODE=<dev-mode-secret-name>:latest
API_GATEWAY_USERINFO_HEADER=<api-gateway-userinfo-header-secret-name>:latest
PG_MAX_CONNS=<pg-max-conns-secret-name>:latest
```

当前 Cloud Run 自动扩展配置：

```text
min-instances: 1
max-instances: 10
concurrency: 40
cpu: 1
memory: 512Mi
```

说明：

- `min-instances=1` 保留一个热实例，减少冷启动，但会产生固定运行成本。
- `max-instances=10` 控制成本并保护 Supabase 连接数。
- `concurrency=40` 比默认值更保守，让服务在单实例压力过高前更早横向扩展。
- 数据库连接预算约等于 `max-instances * PG_MAX_CONNS`。当前建议值是 `10 * 5 = 50`。

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

本地 `.env` 只用于本地运行和初始化 / 更新 Secret Manager。Cloud Run 当前不直接读取 `.env`，也不通过 `--env-vars-file` 写入明文 env。

当前 `cmd/server` 读取以下字段：

```text
DATABASE_URL
PUBLIC_ASSET_BASE_URL
API_ADDR
DEV_MODE
API_GATEWAY_USERINFO_HEADER
PG_MAX_CONNS
```

测试部署建议本地 `.env` 显式包含：

```dotenv
DATABASE_URL=postgresql://...
PUBLIC_ASSET_BASE_URL=<public-asset-base-url>
API_ADDR=:8080
DEV_MODE=true
API_GATEWAY_USERINFO_HEADER=X-Apigateway-Api-Userinfo
PG_MAX_CONNS=5
```

说明：

- `.env` 中的所有运行时字段都只用于创建 / 更新 Secret Manager，不直接写入 Cloud Run 明文 env。
- 当前测试部署仍然是 `DEV_MODE=true`；该值应写入 `<dev-mode-secret-name>` 的 latest 版本。
- 如果 `.env` 没有 `DEV_MODE=true`，或 `<dev-mode-secret-name>` latest 不是 `true`，部署出来的服务就不应是 DEV_MODE。
- `PG_MAX_CONNS` 是单个 Cloud Run 实例的 PostgreSQL 连接池上限。数据库连接预算约等于 `max-instances * PG_MAX_CONNS`。
- 每次重走部署流程、改 `.env`、或要让 GitHub Actions 使用新的运行时配置时，都必须先执行下一节的 `.env` -> Secret Manager 同步。GitHub Actions 不读取 `.env`，只保留 Cloud Run 对 Secret Manager 的引用。

部署前可用下面命令只检查 key 是否存在：

```bash
for key in DATABASE_URL PUBLIC_ASSET_BASE_URL API_ADDR DEV_MODE API_GATEWAY_USERINFO_HEADER PG_MAX_CONNS; do
  awk -F= -v key="$key" '$1==key && substr($0,index($0,"=")+1)!="" { found=1 } END { exit found ? 0 : 1 }' .env \
    || { echo "$key missing in .env"; exit 1; }
done
```

## 6. Secret 准备

不要把运行时配置明文写进部署命令。用本地 `.env` 创建或更新 Secret Manager：

```bash
create_or_update_secret() {
  local env_name="$1"
  local secret_name="$2"
  local value

  value="$(awk -F= -v key="$env_name" '$1==key {print substr($0,index($0,"=")+1); exit}' .env)"
  if [ -z "$value" ]; then
    echo "$env_name missing in .env" >&2
    exit 1
  fi

  if gcloud secrets describe "$secret_name" >/dev/null 2>&1; then
    printf '%s' "$value" | gcloud secrets versions add "$secret_name" --data-file=-
  else
    printf '%s' "$value" | gcloud secrets create "$secret_name" \
      --replication-policy=automatic \
      --data-file=-
  fi
}

create_or_update_secret DATABASE_URL <database-url-secret-name>
create_or_update_secret PUBLIC_ASSET_BASE_URL <public-asset-base-url-secret-name>
create_or_update_secret API_ADDR <api-addr-secret-name>
create_or_update_secret DEV_MODE <dev-mode-secret-name>
create_or_update_secret API_GATEWAY_USERINFO_HEADER <api-gateway-userinfo-header-secret-name>
create_or_update_secret PG_MAX_CONNS <pg-max-conns-secret-name>
```

这一步会为已有 secret 新增 latest version；Cloud Run 引用的是 `:latest`，但为了让所有实例稳定读取新版本，更新 secret 后仍建议重新部署或执行一次 `gcloud run services update` 创建新 revision。

给 Cloud Run 默认运行服务账号读取 secrets 的权限：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
RUN_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

for secret_name in \
  <database-url-secret-name> \
  <public-asset-base-url-secret-name> \
  <api-addr-secret-name> \
  <dev-mode-secret-name> \
  <api-gateway-userinfo-header-secret-name> \
  <pg-max-conns-secret-name>; do
  gcloud secrets add-iam-policy-binding "$secret_name" \
    --member="serviceAccount:${RUN_SERVICE_ACCOUNT}" \
    --role='roles/secretmanager.secretAccessor'
done
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

gcloud run deploy <cloud-run-service-name> \
  --image "$IMAGE" \
  --region <gcp-region> \
  --allow-unauthenticated \
  --port 8080 \
  --min-instances 1 \
  --max-instances 10 \
  --concurrency 40 \
  --cpu 1 \
  --memory 512Mi \
  --set-secrets DATABASE_URL=<database-url-secret-name>:latest,PUBLIC_ASSET_BASE_URL=<public-asset-base-url-secret-name>:latest,API_ADDR=<api-addr-secret-name>:latest,DEV_MODE=<dev-mode-secret-name>:latest,API_GATEWAY_USERINFO_HEADER=<api-gateway-userinfo-header-secret-name>:latest,PG_MAX_CONNS=<pg-max-conns-secret-name>:latest
```

成功输出类似：

```text
Service [<cloud-run-service-name>] revision [<cloud-run-service-name>-00001-tm8] has been deployed and is serving 100 percent of traffic.
Service URL: <cloud-run-service-url>
```

同一个 service name `<cloud-run-service-name>` 重新部署时会创建新 revision，但 service URL 保持稳定。

如果只想更新现有服务的自动扩展配置，不重新构建镜像：

```bash
gcloud run services update <cloud-run-service-name> \
  --region <gcp-region> \
  --min-instances 1 \
  --max-instances 10 \
  --concurrency 40 \
  --cpu 1 \
  --memory 512Mi \
  --update-secrets PG_MAX_CONNS=<pg-max-conns-secret-name>:latest
```

这个命令会创建一个新 revision，但使用现有镜像。`PG_MAX_CONNS` 只有在镜像内的应用代码已经支持该环境变量后才会限制数据库连接池。

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

预期所有运行时配置都通过 `valueFrom.secretKeyRef` 引用 Secret Manager；不要出现明文 `value`。

确认自动扩展配置：

```bash
gcloud run services describe <cloud-run-service-name> \
  --region <gcp-region> \
  --format='yaml(spec.template.metadata.annotations,spec.template.spec.containerConcurrency,spec.template.spec.containers[0].resources,status.latestReadyRevisionName,status.traffic)'
```

预期包含：

```text
autoscaling.knative.dev/minScale: '1'
autoscaling.knative.dev/maxScale: '10'
containerConcurrency: 40
cpu: '1'
memory: 512Mi
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

gcloud builds submit \
  --region <gcp-region> \
  --tag "$IMAGE" \
  .

gcloud run deploy <cloud-run-service-name> \
  --image "$IMAGE" \
  --region <gcp-region> \
  --allow-unauthenticated \
  --port 8080 \
  --min-instances 1 \
  --max-instances 10 \
  --concurrency 40 \
  --cpu 1 \
  --memory 512Mi \
  --set-secrets DATABASE_URL=<database-url-secret-name>:latest,PUBLIC_ASSET_BASE_URL=<public-asset-base-url-secret-name>:latest,API_ADDR=<api-addr-secret-name>:latest,DEV_MODE=<dev-mode-secret-name>:latest,API_GATEWAY_USERINFO_HEADER=<api-gateway-userinfo-header-secret-name>:latest,PG_MAX_CONNS=<pg-max-conns-secret-name>:latest
```

## 13. 绑定 Cloudflare 域名到 Load Balancer

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

当前推荐链路：

```text
<api-domain>
  -> Cloudflare DNS only
  -> Google external Application Load Balancer
  -> Serverless NEG
  -> Cloud Run <cloud-run-service-name>
```

这条链路比 Cloud Run direct domain mapping 多一层入口，但后续切到 API Gateway、接 Cloud Armor、固定 IP、按路径路由和回滚都更方便。

### 13.1 当前 Cloudflare DNS 保护边界

绑定 API 域名时不要修改以下记录：

```text
<root-domain> A
www.<root-domain> CNAME
<root-domain> MX
<root-domain> TXT
sig1._domainkey.<root-domain> CNAME
```

当前 API 只需要维护两条 DNS 记录：

```text
<api-domain> A <lb-static-ip>
_acme-challenge.<api-domain> CNAME <certificate-manager-dns-auth-target>
```

并且保持：

```text
proxied=false
DNS only / 灰云
```

证书签发完成前不要打开 Cloudflare 代理。Cloudflare 代理可能影响 Google managed certificate 的验证和续期。

### 13.2 一次性启用 API

Load Balancer + Certificate Manager 至少需要：

```bash
gcloud services enable \
  compute.googleapis.com \
  certificatemanager.googleapis.com \
  networkservices.googleapis.com
```

### 13.3 创建 Certificate Manager DNS authorization

用 DNS authorization 可以先签发证书，再切正式 `A` 记录，避免直接改流量后等待证书。

```bash
gcloud certificate-manager dns-authorizations create <certificate-dns-authorization-name> \
  --location=global \
  --domain=<api-domain>

gcloud certificate-manager dns-authorizations describe <certificate-dns-authorization-name> \
  --location=global \
  --format='yaml(dnsResourceRecord)'
```

把返回的 DNS 记录加到 Cloudflare：

```text
Type: CNAME
Name: _acme-challenge.api
Target: <certificate-manager-dns-auth-target>
Proxy status: DNS only
TTL: Auto
```

确认解析：

```bash
dig +short _acme-challenge.<api-domain> CNAME
```

### 13.4 创建证书和 certificate map

```bash
gcloud certificate-manager certificates create <certificate-name> \
  --location=global \
  --domains=<api-domain> \
  --dns-authorizations=<certificate-dns-authorization-name>

gcloud certificate-manager maps create <certificate-map-name> \
  --location=global

gcloud certificate-manager maps entries create <certificate-map-entry-name> \
  --location=global \
  --map=<certificate-map-name> \
  --hostname=<api-domain> \
  --certificates=<certificate-name>
```

检查证书状态：

```bash
gcloud certificate-manager certificates describe <certificate-name> \
  --location=global \
  --format='yaml(managed.state,managed.authorizationAttemptInfo)'
```

等到：

```text
managed.state: ACTIVE
authorizationAttemptInfo.state: AUTHORIZED
```

### 13.5 创建 Load Balancer 到 Cloud Run

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
CERT_MAP_PATH="projects/${PROJECT_ID}/locations/global/certificateMaps/<certificate-map-name>"

gcloud compute addresses create <lb-ip-name> \
  --global \
  --ip-version=IPV4

gcloud compute network-endpoint-groups create <serverless-neg-name> \
  --region=<gcp-region> \
  --network-endpoint-type=serverless \
  --cloud-run-service=<cloud-run-service-name>

gcloud compute backend-services create <backend-service-name> \
  --global \
  --load-balancing-scheme=EXTERNAL_MANAGED \
  --protocol=HTTP

gcloud compute backend-services add-backend <backend-service-name> \
  --global \
  --network-endpoint-group=<serverless-neg-name> \
  --network-endpoint-group-region=<gcp-region>

gcloud compute url-maps create <url-map-name> \
  --default-service=<backend-service-name>

gcloud compute target-https-proxies create <https-proxy-name> \
  --global \
  --url-map=<url-map-name> \
  --certificate-map="$CERT_MAP_PATH"

gcloud compute forwarding-rules create <https-forwarding-rule-name> \
  --global \
  --load-balancing-scheme=EXTERNAL_MANAGED \
  --network-tier=PREMIUM \
  --address=<lb-ip-name> \
  --target-https-proxy=<https-proxy-name> \
  --ports=443
```

取静态 IP：

```bash
gcloud compute addresses describe <lb-ip-name> \
  --global \
  --format='value(address)'
```

### 13.6 切 Cloudflare 正式 DNS

证书 `ACTIVE` 后，先用 `curl --resolve` 预验证 LB：

```bash
curl -i \
  --resolve <api-domain>:443:<lb-static-ip> \
  https://<api-domain>/api/me
```

预期未带 token 返回：

```text
401 unauthorized
```

确认后，把 Cloudflare 的 API 记录切成：

```text
Type: A
Name: api
Content: <lb-static-ip>
Proxy status: DNS only
TTL: Auto
Comment: Google external Application Load Balancer for Cloud Run API
```

保留 `_acme-challenge.<api-domain>` CNAME。它用于 Google managed certificate 的授权和续期。

DNS 验证：

```bash
dig +short <api-domain> A
dig +short _acme-challenge.<api-domain> CNAME
```

最终链路验证：

```bash
curl -i https://<api-domain>/api/me
```

预期：

```text
HTTP/2 401
via: 1.1 google
```

`via: 1.1 google` 说明请求经过 Google Load Balancer。

### 13.7 状态检查和回滚

查看 LB 入口：

```bash
gcloud compute forwarding-rules describe <https-forwarding-rule-name> \
  --global \
  --format='yaml(IPAddress,loadBalancingScheme,portRange,target)'
```

查看证书：

```bash
gcloud certificate-manager certificates describe <certificate-name> \
  --location=global \
  --format='yaml(managed.state,managed.authorizationAttemptInfo)'
```

如果需要回滚到 Cloud Run direct domain mapping，前提是原 Cloud Run domain mapping 仍然存在且 Ready。把 Cloudflare API 记录改回：

```text
Type: CNAME
Name: api
Target: ghs.googlehosted.com
Proxy status: DNS only
```

也可以临时直接使用 Cloud Run 默认 `run.app` HTTPS endpoint 绕过自定义域名。

### 13.8 以后是否打开 Cloudflare 代理

测试阶段建议继续保持 DNS only。

如果之后要打开 Cloudflare 橙云代理，先确认 Load Balancer 链路可用且 Certificate Manager 证书已经 `ACTIVE`，然后：

- Cloudflare DNS record 改为 proxied。
- Cloudflare SSL/TLS mode 使用 `Full (strict)`。
- 保留 `_acme-challenge.<api-domain>` DNS only。
- 避免启用会干扰 Google certificate renewal 的强制跳转规则，尤其是证书验证路径相关规则。

## 14. GitHub Actions 自动部署

当前推荐自动部署边界：

```text
GitHub Actions 只负责构建镜像和部署镜像。
Cloud Run 运行时配置全部来自 Secret Manager。
GitHub 不保存 DATABASE_URL 或其他业务 secret 明文。
```

因此，GitHub Actions 触发前如果运行时配置有变化，先在本地更新 `.env`，再执行第 6 节同步到 Secret Manager。workflow 本身不会读取 `.env`，也不会把 `.env` 推送或上传。当前测试环境的 `DEV_MODE=true` 由 `<dev-mode-secret-name>:latest` 控制。

### 14.1 GCP deploy 身份

创建 GitHub Actions 专用 service account：

```bash
gcloud iam service-accounts create <github-actions-service-account-name> \
  --display-name="GitHub Actions Cloud Run Deployer"
```

给 deploy service account 授权：

```bash
PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"
GITHUB_ACTIONS_SERVICE_ACCOUNT="<github-actions-service-account-email>"
RUN_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${GITHUB_ACTIONS_SERVICE_ACCOUNT}" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${GITHUB_ACTIONS_SERVICE_ACCOUNT}" \
  --role="roles/cloudbuild.builds.editor"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${GITHUB_ACTIONS_SERVICE_ACCOUNT}" \
  --role="roles/storage.admin"

gcloud iam service-accounts add-iam-policy-binding "$RUN_SERVICE_ACCOUNT" \
  --member="serviceAccount:${GITHUB_ACTIONS_SERVICE_ACCOUNT}" \
  --role="roles/iam.serviceAccountUser"
```

说明：

- `roles/run.admin` 用于创建 Cloud Run revision 和切流量。
- `roles/cloudbuild.builds.editor` 用于提交 Cloud Build。
- `roles/storage.admin` 用于 `gcloud builds submit` 上传源码包。
- `roles/iam.serviceAccountUser` 允许部署时使用 Cloud Run runtime service account。

### 14.2 Workload Identity Federation

创建 GitHub OIDC provider，并用 attribute condition 限制到当前 repository 和 `main` 分支：

```bash
gcloud iam workload-identity-pools create <workload-identity-pool-id> \
  --location=global \
  --display-name="GitHub Actions"

gcloud iam workload-identity-pools providers create-oidc <workload-identity-provider-id> \
  --location=global \
  --workload-identity-pool=<workload-identity-pool-id> \
  --display-name="GitHub OIDC" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner,attribute.ref=assertion.ref" \
  --attribute-condition="attribute.repository=='<github-owner>/<github-repo>' && attribute.ref=='refs/heads/main'"
```

允许该 repository impersonate deploy service account：

```bash
PROJECT_NUMBER="$(gcloud projects describe "$(gcloud config get-value project 2>/dev/null)" --format='value(projectNumber)')"
MEMBER="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/<workload-identity-pool-id>/attribute.repository/<github-owner>/<github-repo>"

gcloud iam service-accounts add-iam-policy-binding <github-actions-service-account-email> \
  --member="$MEMBER" \
  --role="roles/iam.workloadIdentityUser"
```

### 14.3 GitHub Actions secrets

用 GitHub encrypted secrets 保存部署身份标识。它们不是业务 secret，但放入 GitHub secrets 可以减少 workflow 文件暴露的信息：

```bash
printf '%s' '<workload-identity-provider-resource-name>' | \
  gh secret set GCP_WORKLOAD_IDENTITY_PROVIDER --repo <github-owner>/<github-repo>

printf '%s' '<github-actions-service-account-email>' | \
  gh secret set GCP_SERVICE_ACCOUNT --repo <github-owner>/<github-repo>

printf '%s' '<gcp-project-id>' | \
  gh secret set GCP_PROJECT_ID --repo <github-owner>/<github-repo>

printf '%s' '<gcp-region>' | \
  gh secret set GCP_REGION --repo <github-owner>/<github-repo>

printf '%s' '<cloud-run-service-name>' | \
  gh secret set CLOUD_RUN_SERVICE --repo <github-owner>/<github-repo>
```

### 14.4 Workflow 文件

仓库使用：

```text
.github/workflows/deploy-cloud-run.yml
```

该 workflow 做：

1. `make check`
2. 使用 GitHub OIDC 通过 Workload Identity Federation 登录 GCP
3. `gcloud builds submit` 构建镜像
4. `gcloud run deploy` 部署镜像并保留 autoscaling 参数

workflow 不读取 `.env`，不传 `--env-vars-file`，不保存业务配置。Cloud Run env 由 Secret Manager 引用保持。

workflow 中的 `gcloud builds submit` 使用 `--suppress-logs`。原因是 GitHub Actions deploy service account 只负责提交构建和部署，不授予项目级 Viewer/Owner 来 stream Cloud Build logs；这样可以减少 GitHub Actions 日志里暴露的 GCP 细节。若 Cloud Build 失败，用本地已授权账号或 GCP Console 查看对应 build logs。

### 14.5 自动部署验证

手动触发或 push 到 `main` 后检查：

```bash
gh run list --workflow deploy-cloud-run.yml --limit 5
```

如果失败，查看具体日志：

```bash
gh run view <run-id> --log-failed
```

成功后确认 Cloud Run 新 revision 和自定义域名：

```bash
gcloud run services describe <cloud-run-service-name> \
  --region <gcp-region> \
  --format='yaml(status.latestReadyRevisionName,status.traffic)'

curl -i https://<api-domain>/api/me
```

未带 token 时返回 `401 unauthorized`，且响应包含 `via: 1.1 google`，表示 Load Balancer 到 Cloud Run 链路正常。

## 15. 生产化前必须调整

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
