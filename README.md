# 모니터링 에이전트(Metric Agent) 가이드

본 에이전트는 기업의 서버 자원과 컨테이너화된 애플리케이션의 상태를 OpenTelemetry 표준 규격으로 수집하는 경량 소프트웨어입니다.

## 상세 수집 항목 안내

설치 전, 본 에이전트가 수집하는 데이터의 종류와 기술적 명세를 확인하십시오. 모든 데이터는 OTLP/HTTP 프로토콜을 통해 안전하게 전송됩니다.

### 1. 서버 자원 지표 (Host Metrics)
서버의 전체적인 부하 상태를 확인하기 위해 리눅스 /proc 파일시스템 및 시스템 호출을 통해 데이터를 추출합니다.
- CPU 사용률: 서버 전체 프로세서의 순간 사용량 (%)
- 메모리(RAM) 사용률: 전체 물리 메모리 대비 실제 사용 중인 메모리 비율 (%)
- 디스크 사용률: 루트(/) 파티션의 전체 용량 대비 사용량 (%)
- 네트워크 트래픽: 주요 네트워크 인터페이스의 수신/송신 누적 바이트 (bytes)

### 2. 애플리케이션 지표 (Docker Metrics)
Docker로 운영 중인 서비스의 경우, 리눅스 cgroup v2 및 Docker API를 통해 정밀한 수집을 수행합니다.
- 컨테이너 CPU 사용량: 컨테이너 생성 이후 누적된 CPU 사용 시간 (단위: nanoseconds)
- 컨테이너 메모리 사용량: 해당 컨테이너가 점유 중인 실제 메모리 양 (단위: bytes)
- 컨테이너 네트워크 트래픽: 개별 컨테이너의 수신/송신 누적 바이트 (단위: bytes)
- 컨테이너 식별 정보: 컨테이너 이름, 고유 ID (12자리)

### 3. 로그 수집 (Log Collection)
시스템 장애 진단 및 애플리케이션 추적을 위해 실시간 로그를 수집합니다.
- 시스템 로그: /var/log/syslog 또는 /var/log/messages 파일의 실시간 테일링 로그
- 컨테이너 로그: Docker API 로그 스트림을 통한 실시간 표준 출력(stdout/stderr) 메시지
- 데이터 정제: 로그 전송 시 Docker 특유의 8바이트 바이너리 헤더를 제거한 순수 텍스트 본문 전송

## 방법 1. Docker 이미지로 실행 (권장)

서버에 Docker가 설치되어 있는 경우, 이미지를 내려받아 즉시 실행이 가능합니다.

1. 이미지 다운로드
```bash
sudo docker pull kimhongseok/metric-agent:latest
```

2. 에이전트 실행
```bash
sudo docker run -d \
  --name metric-agent \
  --restart always \
  --privileged \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/log:/var/log \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -e MONITORING_ID="귀사의_고유_코드" \
  -e COLLECTOR_URL="수집_서버_주소:4318" \
  kimhongseok/metric-agent:latest
```

## 방법 2. curl 명령어로 직접 설치

서버에 직접 실행 파일을 내려받아 구동하는 방식입니다.

1. 에이전트 다운로드 및 권한 부여
```bash
curl -fLO http://agent.clearplate.store/metric-agent
chmod +x metric-agent
```

2. 환경 변수 설정 및 실행
```bash
export MONITORING_ID="귀사의_고유_코드"
export COLLECTOR_URL="수집_서버_주소:4318"
sudo -E ./metric-agent
```

## 준비 사항
- 운영체제: Linux (Ubuntu 20.04 이상 권장)
- 네트워크: 수집 서버의 4318 포트(HTTP)로 아웃바운드 통신이 허용되어야 합니다.
- 자동 감지: Docker 서비스가 실행 중인 경우 별도 설정 없이 컨테이너 모니터링 기능이 활성화됩니다.

## 주요 특징
- 표준 프로토콜: 글로벌 표준인 OpenTelemetry(OTLP) 기반으로 데이터의 범용성이 높습니다.
- 자동 식별: 기업 코드와 서버 호스트네임을 조합하여 관리 대시보드에 자동 분류됩니다.
- 리소스 최적화: Go 언어로 빌드된 단일 바이너리로 작동하여 시스템 부하를 최소화합니다.
