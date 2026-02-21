# 1단계: 빌드 스테이지
FROM golang:1.24-alpine AS builder

# 빌드에 필요한 도구 설치
RUN apk add --no-cache git

# 작업 디렉토리 설정
WORKDIR /app

# 의존성 파일 복사 및 다운로드
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# 정적 바이너리 빌드 (CGO_ENABLED=0)
RUN CGO_ENABLED=0 GOOS=linux go build -o metric-agent main.go

# 2단계: 실행 스테이지
FROM alpine:latest

# 로그 수집 및 Docker API 통신을 위한 라이브러리 설치
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 빌드된 바이너리 복사
COPY --from=builder /app/metric-agent .

# 에이전트 실행
# 호스트의 로그와 지표를 읽기 위해 root 권한이 필요할 수 있습니다.
ENTRYPOINT ["./metric-agent"]
