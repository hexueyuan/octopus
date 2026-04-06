#!/bin/bash
set -e

# =============================================================================
# 配置区 - 根据实际环境修改
# =============================================================================

IMAGE_NAME="hexueyuan/octopus"
IMAGE_TAG="latest"
CONTAINER_NAME="octopus"
BACKUP_NAME="octopus-backup"

# 宿主机数据目录 -> 容器 /app/data
DATA_VOLUME="/Users/hexueyuan/Workroot/service/bestrui/octopus"

# 端口映射: 宿主机端口 -> 容器端口
HOST_PORT=8086
CONTAINER_PORT=8080

# Go 环境 - 优先使用 Homebrew 安装的 Go (需要 >= 1.24)
HOMEBREW_GO="/opt/homebrew/bin/go"

# 构建目标平台
TARGET_OS="linux"
TARGET_ARCH="arm64"

# =============================================================================
# 内部变量
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOCKERFILE="${PROJECT_DIR}/scripts/dockerfiles/Dockerfile.alpine"
DOCKER_PLATFORM="${TARGET_OS}/${TARGET_ARCH}"
BUILD_BIN_DIR="${PROJECT_DIR}/build/docker/${DOCKER_PLATFORM}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()    { echo -e "${CYAN}[INFO]${NC}  $1"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_step()    { echo -e "\n${CYAN}==>${NC} $1"; }

# =============================================================================
# 环境检查
# =============================================================================

setup_go() {
    log_step "配置 Go 环境"

    # 清除环境中可能存在的旧 GOROOT，避免干扰 `go env GOROOT` 的输出
    unset GOROOT

    if [ -x "${HOMEBREW_GO}" ]; then
        GO_BIN="${HOMEBREW_GO}"
        GO_ROOT="$(${GO_BIN} env GOROOT)"
    elif command -v go &>/dev/null; then
        GO_BIN="$(command -v go)"
        GO_ROOT="$(${GO_BIN} env GOROOT)"
    else
        log_error "未找到 Go，请先安装 Go >= 1.24"
        exit 1
    fi

    GO_VERSION="$(${GO_BIN} version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)"
    GO_MAJOR="$(echo "${GO_VERSION}" | grep -oE '[0-9]+\.[0-9]+' | cut -d. -f1)"
    GO_MINOR="$(echo "${GO_VERSION}" | grep -oE '[0-9]+\.[0-9]+' | cut -d. -f2)"

    if [ "${GO_MAJOR}" -lt 1 ] || ([ "${GO_MAJOR}" -eq 1 ] && [ "${GO_MINOR}" -lt 24 ]); then
        log_error "Go 版本过低: ${GO_VERSION}，需要 >= 1.24"
        log_error "当前路径: ${GO_BIN}"
        exit 1
    fi

    log_success "Go ${GO_VERSION} (${GO_BIN})"
}

check_deps() {
    log_step "检查依赖"

    local missing=()
    command -v docker &>/dev/null || missing+=("docker")
    command -v node   &>/dev/null || missing+=("node")
    command -v pnpm   &>/dev/null || missing+=("pnpm")

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "缺少依赖: ${missing[*]}"
        exit 1
    fi

    log_success "docker $(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
    log_success "node $(node --version)"
    log_success "pnpm $(pnpm --version)"
}

# =============================================================================
# 构建
# =============================================================================

build_frontend() {
    log_step "构建前端"
    cd "${PROJECT_DIR}/web"
    pnpm install --frozen-lockfile 2>&1 | tail -1
    pnpm run build 2>&1 | tail -5
    cd "${PROJECT_DIR}"

    rm -rf static/out
    mv web/out static/
    log_success "前端构建完成 -> static/out"
}

build_backend() {
    log_step "编译后端 (${TARGET_OS}/${TARGET_ARCH})"

    local git_version commit_id build_time ldflags
    git_version="$(git -C "${PROJECT_DIR}" describe --tags --abbrev=0 2>/dev/null || echo 'dev')"
    commit_id="$(git -C "${PROJECT_DIR}" rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
    build_time="$(TZ='Asia/Shanghai' date +'%F %T %z')"

    ldflags="-X 'github.com/bestruirui/octopus/internal/conf.Version=${git_version}'"
    ldflags+=" -X 'github.com/bestruirui/octopus/internal/conf.BuildTime=${build_time}'"
    ldflags+=" -X 'github.com/bestruirui/octopus/internal/conf.Author=bestrui'"
    ldflags+=" -X 'github.com/bestruirui/octopus/internal/conf.Commit=${commit_id}'"
    ldflags+=" -s -w"

    mkdir -p "${BUILD_BIN_DIR}"

    GOROOT="${GO_ROOT}" GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" CGO_ENABLED=0 \
        "${GO_BIN}" build \
        -o "${BUILD_BIN_DIR}/octopus" \
        -ldflags="${ldflags}" \
        -tags=jsoniter \
        "${PROJECT_DIR}/" 2>&1

    local size
    size="$(du -h "${BUILD_BIN_DIR}/octopus" | cut -f1 | xargs)"
    log_success "编译完成: ${BUILD_BIN_DIR}/octopus (${size})"
}

build_image() {
    log_step "构建 Docker 镜像 ${IMAGE_NAME}:${IMAGE_TAG}"

    docker build \
        -t "${IMAGE_NAME}:${IMAGE_TAG}" \
        --build-arg TARGETPLATFORM="${DOCKER_PLATFORM}" \
        -f "${DOCKERFILE}" \
        "${PROJECT_DIR}" 2>&1 | tail -3

    log_success "镜像构建完成: ${IMAGE_NAME}:${IMAGE_TAG}"
}

# =============================================================================
# 部署
# =============================================================================

deploy() {
    log_step "部署容器"

    # 清理上次残留的备份容器
    if docker ps -a --format '{{.Names}}' | grep -q "^${BACKUP_NAME}$"; then
        log_warn "发现残留备份容器 ${BACKUP_NAME}，正在清理"
        docker rm -f "${BACKUP_NAME}" &>/dev/null || true
    fi

    # 检查当前是否有运行中的容器
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_info "停止旧容器并备份为 ${BACKUP_NAME}"
        # 一条命令完成: 停止 -> 备份 -> 启动新容器，减少服务中断时间
        docker stop "${CONTAINER_NAME}" && \
        docker rename "${CONTAINER_NAME}" "${BACKUP_NAME}" && \
        docker run -d \
            --name "${CONTAINER_NAME}" \
            -v "${DATA_VOLUME}:/app/data" \
            -p "${HOST_PORT}:${CONTAINER_PORT}" \
            --restart unless-stopped \
            "${IMAGE_NAME}:${IMAGE_TAG}"
    else
        log_info "未发现旧容器，直接启动"
        docker run -d \
            --name "${CONTAINER_NAME}" \
            -v "${DATA_VOLUME}:/app/data" \
            -p "${HOST_PORT}:${CONTAINER_PORT}" \
            --restart unless-stopped \
            "${IMAGE_NAME}:${IMAGE_TAG}"
    fi

    # 等待容器启动并检查状态
    sleep 2
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log_success "容器启动成功"
        docker ps --filter "name=${CONTAINER_NAME}" --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
    else
        log_error "容器启动失败，查看日志:"
        docker logs "${CONTAINER_NAME}" 2>&1 | tail -20
        echo ""
        log_warn "正在自动回滚..."
        rollback
        exit 1
    fi
}

# =============================================================================
# 回滚
# =============================================================================

rollback() {
    log_step "回滚到备份容器"

    if ! docker ps -a --format '{{.Names}}' | grep -q "^${BACKUP_NAME}$"; then
        log_error "未找到备份容器 ${BACKUP_NAME}，无法回滚"
        exit 1
    fi

    # 停止并删除新容器
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker stop "${CONTAINER_NAME}" &>/dev/null || true
        docker rm "${CONTAINER_NAME}" &>/dev/null || true
    fi

    docker rename "${BACKUP_NAME}" "${CONTAINER_NAME}"
    docker start "${CONTAINER_NAME}"

    log_success "回滚完成"
    docker ps --filter "name=${CONTAINER_NAME}" --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
}

# =============================================================================
# 清理备份
# =============================================================================

cleanup() {
    log_step "清理备份"

    if docker ps -a --format '{{.Names}}' | grep -q "^${BACKUP_NAME}$"; then
        docker rm "${BACKUP_NAME}"
        log_success "已删除备份容器 ${BACKUP_NAME}"
    else
        log_info "没有需要清理的备份容器"
    fi
}

# =============================================================================
# 入口
# =============================================================================

usage() {
    cat <<EOF
用法: $0 <命令>

命令:
  all       完整流程: 构建前端 + 编译后端 + 构建镜像 + 部署
  build     仅构建: 构建前端 + 编译后端 + 构建镜像 (不部署)
  deploy    仅部署: 用已有镜像更新容器
  rollback  回滚到备份容器
  cleanup   确认新版本稳定后，删除备份容器

示例:
  $0 all        # 一键编译部署
  $0 rollback   # 出问题时回滚
  $0 cleanup    # 确认没问题后清理备份
EOF
}

main() {
    cd "${PROJECT_DIR}"

    case "${1:-}" in
        all)
            check_deps
            setup_go
            build_frontend
            build_backend
            build_image
            deploy
            echo ""
            log_success "全部完成! 如需回滚: $0 rollback"
            ;;
        build)
            check_deps
            setup_go
            build_frontend
            build_backend
            build_image
            echo ""
            log_success "构建完成! 使用 '$0 deploy' 部署"
            ;;
        deploy)
            deploy
            echo ""
            log_success "部署完成! 如需回滚: $0 rollback"
            ;;
        rollback)
            rollback
            ;;
        cleanup)
            cleanup
            ;;
        *)
            usage
            exit 1
            ;;
    esac
}

main "$@"
