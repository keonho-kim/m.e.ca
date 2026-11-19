# 최적화 구현 전략

[문서 목록](README.md) · [기능별 최적화](optimization.md) · [개발 가이드](development.md)

현재 동작을 담당 코드와 연결한다. 설정값과 공개 계약은 각 기능 가이드에서 확인한다.

## 모델 문맥과 Provider

실행이 보유한 문맥 예산 안에서 SQLite의 선택 필드를 복원한다. SQLite는 원문 페이로드를 가져오기 전에 선택한 필드의 크기를 확인한다. 비전 입력을 지원하지 않는 경우 이미지 본문을 건너뛰고 생략 안내만 포함한다. 압축은 최근 native 호출·결과 묶음과 아직 읽지 않은 관찰을 보존하면서 요약을 새 리비전으로 확정한다. [문맥 복원](../internal/storage/sqlite/context.go) · [문맥 예약](../internal/agent/run/context_memory.go) · [압축](../internal/agent/compact/controller.go)

문맥 투영은 같은 실행의 변화 없는 메시지를 재사용한다. 모델 선택·마스킹 기준·메시지와 도구 짝이 달라진 구간은 다시 구성한다. Provider 인코더는 변경되지 않은 앞부분의 직렬화 결과를 제한된 용량으로 보유한다. 아직 모델이 읽지 않은 브라우저 관찰 앞에서 재사용 범위를 끝내고 실행이 끝나면 캐시를 해제한다. [런타임의 모델 요청](runtime.md#모델-요청-준비) · [문맥 투영](../internal/agent/context/projection.go) · [인코딩 재사용](../internal/provider/protocol/source_history.go)

vLLM 토큰화 응답은 바이트 한도 안에서 스트리밍으로 읽고 입력 토큰 수와 컨텍스트 한도를 추출한다. 사용하지 않는 토큰 ID 배열은 건너뛴다. 압축은 마스킹·발췌한 레코드를 직렬화하기 전에 마지막 선택 항목과 비교하고, 선택한 항목의 순서와 사용자 지시는 보존한다. [토큰화 응답](../internal/provider/httpclient/context.go) · [압축 원본 선택](../internal/agent/compact/source.go)

## 대화 저장과 SQLite

저널은 준비 버퍼 범위에서 레코드를 쓰기 트랜잭션 전에 인코딩한다. 큰 본문이나 공유 대상은 참조 가능한 페이로드로 저장하고, 본문과 이를 참조하는 기록은 한 트랜잭션으로 확정한다. 브라우저 텍스트 결과의 모델용 문자열과 화면용 JSON은 같은 본문을 참조할 수 있다. 복원 시 참조·길이·무결성을 확인한다. [쓰기 준비](../internal/storage/sqlite/prepare.go) · [페이로드 분리](../internal/storage/content/pack.go) · [참조 복원](../internal/storage/sqlite/record_codec.go)

모델 문맥, 화면 이력, 원문은 각자 필요한 필드만 요청한다. 화면 조회는 페이로드 메타데이터로 예산을 확인한 뒤 본문을 읽는다. 같은 이력 페이지에서 여러 번 참조하는 본문은 제한된 용량 안에서 재사용하고 마지막 참조가 지나면 해제한다. [세션 저장 계약](configuration.md#세션-저장과-복원) · [이력 투영](../internal/storage/sqlite/timeline.go) · [페이지 단위 복원](../internal/storage/sqlite/page_payloads.go)

단일 텍스트 본문을 참조하는 로컬 모델 메시지는 원본의 길이와 해시, 저장된 메시지의 참조 위치를 확인한 뒤 본문을 직접 채운다. [메시지 본문 복원](../internal/storage/sqlite/context_record.go)

## 팀 실행과 공용 기록

방이 공용 대화와 사용자 이미지 원본을 소유한다. 참여자별 SQLite 기록은 공용 원본의 식별과 검증에 필요한 참조를 보유한다. 복원 시 같은 페이지의 참조를 묶어서 읽고 출처와 무결성을 검사한다. 참여자의 비공개 문맥과 승인 상태는 해당 실행이 소유한다. [팀 저장 계약](team.md#프롬프트와-저장) · [공용 본문 복원](../internal/storage/sqlite/team_context_page.go) · [공용 이미지 복원](../internal/storage/sqlite/team_images.go)

독립 작업은 같은 요청의 예산 안에서 병렬 실행한다. 확인한 변경 범위가 겹치면 예약으로 조율하고 실제 실행 전에 다시 확인한다. 공유 명령 출력 예산은 팀 전체에서 동시에 복원하는 본문의 양을 제한한다. [팀 변경 범위](../internal/tools/team/workspace/) · [출력 복원 예산](../internal/tools/local/process/projection.go)

## 셸과 파일 작업

명령 출력은 제한된 캡처에서 수집하고 큰 출력은 임시 디스크로 이동한다. 실행 중에는 바뀐 구간을 읽고, 완료 원문은 기존 캡처를 이용해 저널에 확정한다. 확정 후 소유한 임시 파일을 반환한다. 독립 호출의 병렬 준비는 결과의 원래 순서와 실행 직전의 승인·대상 확인을 지킨다. [명령 수명](runtime.md#오래-실행되는-명령) · [출력 수집](../internal/tools/local/process/capture_file.go) · [완료 원문](../internal/tools/local/command_capture.go) · [병렬 준비](../internal/tools/local/parallel.go)

파일 시스템 감시는 같은 디렉터리의 활성 구독이 공유하고 마지막 구독에서 반환한다. 편집기의 변경 확인은 감시 리비전이 같으면 메타데이터에서 끝내고, 달라졌을 때 내용의 리비전을 비교한다. 저장 시에는 원본을 다시 검사한다. [편집 계약](webui-api.md#완료예약편집) · [공유 감시](../internal/fileview/watch_pool.go) · [변경 확인](../internal/fileview/editor_probe.go)

## 스킬 목록과 참고 파일

기본 스킬을 배포할 때는 현재 스킬 디렉터리를 잠그고 내장 묶음의 디렉터리 충돌을 먼저 검사한다. 누락한 파일만 임시 파일에 쓰고 동기화한 뒤 기존 이름과 충돌하지 않을 때 공개한다. 디렉터리의 항목 변경은 묶음 단위로 동기화한다. [기본 스킬 배포](../internal/config/bootstrap/skills.go) · [원자적 파일 공개](../internal/config/statefs/files.go)

목록 조회는 이름·설명 등 메타데이터를 읽는다. 후속 페이지는 같은 목록 스냅샷을 사용한다. 스냅샷은 마스킹 기준이 바뀌거나 보유 시간·용량 한도에 도달하면 회수한다. 참고 파일은 리비전을 확인한 열린 파일에서 필요한 바이트 범위만 읽고 UTF-8 경계를 맞춘다. [스킬 조회](skills.md#사용자-호출과-모델-도구) · [목록 읽기](../internal/skills/catalog.go) · [목록 보유](../internal/skills/catalog_retention.go) · [부분 읽기](../internal/skills/resources.go)

읽은 지침은 현재 실행이 보유한다. 원래 읽기 결과가 요청에 있으면 다시 넣지 않고, 압축으로 사라진 필요한 지침을 문맥에 연결한다. [문맥과 기록](skills.md#문맥과-기록) · [지침 문맥](../internal/agent/context/skill_reads.go)

## 브라우저 관찰과 세션

도우미의 스냅샷 변경 추적을 이용한다. 도우미가 변경 없음을 보고하고 기준 리비전·조회 조건·주소·마스킹 기준이 일치하면 기존 결과 ID와 현재 커서를 돌려준다. 변경분만 받았거나 기준을 다시 맞춰야 할 때는 전체 스냅샷을 요청한다. 텍스트·오류·스크린샷은 브라우저 주소 확인과 한 호출로 묶는다. [관찰 계약](browser.md#관찰과-컨텍스트) · [도우미 관찰](../internal/tools/browser/inspect.go)

세션은 UTF-8 경계에 맞춘 임시 블록과 현재 이어 읽기 위치를 소유한다. 정확한 결과 ID·리비전·offset으로 읽을 때만 커서를 전진시키고 다 읽은 블록을 삭제한다. 모델 응답이 완료되면 해당 관찰을 짧은 열람 표시로 바꾼다. SQLite는 복원하기 전에 같은 경계를 적용하고, 실행은 현재 세션에서 받은 커서를 압축 뒤에도 별도로 전달한다. [임시 관찰](../internal/tools/browser/paging.go) · [실행 문맥](../internal/agent/run/browser_pages.go) · [SQLite 복원](../internal/storage/sqlite/browser_context.go)

도구 목록을 만들 때는 설치 표시와 실행 경로만 확인하고, 모델 사용 지침은 제공한 도구에 맞춰 구성한다. 세션을 열 때 도우미의 무결성을 검사한다. 관리 Chrome 갱신은 다운로드 잠금과 버전 선택 잠금을 분리하고, 사용 중인 버전은 세션 종료가 확인될 때까지 보호한다. [설치 확인](../internal/tools/browser/runtime/availability.go) · [모델 요청 구성](../internal/agent/run/start.go) · [세션 수명](../internal/tools/browser/session.go) · [관리 설치본](../internal/tools/browser/runtime/managed.go) · [사용 기록](../internal/tools/browser/runtime/chrome_use.go)

### 웹 문서 조회

문서 URL의 DNS 응답을 검사하고 승인된 IP를 같은 출처의 연결과 렌더링 리소스에 재사용한다. HTML은 단독 임시 Chromium에서 읽고, DevTools 파이프로 받은 DOM을 한 번 파싱해 Markdown으로 만든다. 텍스트·Markdown이나 기존 커서는 바로 읽는다. 문서 처리가 끝나면 연결과 브라우저 자원을 반환한다. 연결과 렌더링은 응답·리소스·프로세스 예산을 사용한다. [HTTP 연결](../internal/tools/browser/webfetch/client.go) · [Chromium 수명](../internal/tools/browser/chromium/process_unix.go) · [문서 읽기](../internal/tools/browser/webfetch/document.go)

도구 호출 묶음은 남은 모델 문맥을 나눠 사용한다. 즉시 제공하는 페이지와 읽지 않은 임시 블록의 수명이 다르며, 읽은 페이지는 다음 모델 요청에서 짧은 표시로 바뀐다. 브라우저와 같은 블록 판독 및 정리 구현을 사용한다. [페이지 예산](../internal/agent/run/webfetch_budget.go) · [블록 저장과 회수](../internal/tools/observation/body.go) · [WebFetch 계약](web-fetch.md#이어-읽기)

## 이미지와 미리보기

헤더에서 크기와 형식을 확인하고 디코딩 예산을 먼저 확보한다. 원본은 불변 스냅샷으로 남긴다. 지원 형식은 압축된 바이트를 그대로 전달하고 변환이 필요한 형식에서는 표시·추론에 쓸 대표 이미지를 준비한다. [이미지 입력](images.md#자원과-저장) · [디코딩](../internal/media/imagefile/decode.go) · [디코딩 예산](../internal/processmemory/)

WebUI는 작고 지원되는 첨부를 로컬에서 순차 축소하고 나머지는 서버의 제한된 썸네일 경로를 사용한다. 원본 파일은 모델 요청을 위해 보존하고 카드가 제거되면 미리보기 URL을 해제한다. 저장 이미지의 썸네일은 접근과 원본 상태를 확인한 뒤 가져오며, 같은 원본의 동시 변환을 합친다. [미리보기 계약](webui-api.md#첨부-미리보기와-정적-자산) · [브라우저 축소](../webui/src/shared/lib/image-preview/) · [서버 썸네일 캐시](../internal/serverapi/thumbnail.go)

## 서버 이벤트와 원문 조회

서버는 작업 수명과 구독 수명을 분리한다. 실행 사건에서 만든 순서 있는 변경분과 제한된 스냅샷을 전송하며 동시 관찰자가 있는 동안 정제된 사건과 JSON을 공유한다. 내장 WebUI는 작업·파일 구독을 한 WebSocket에 담고 ACK 창으로 미확인 프레임 수를 제한한다. 재접속은 관찰 위치와 작업 스냅샷을 복구한다. [이벤트 계약](server.md#이벤트재접속수명) · [변경분](../internal/commands/server/stream_projection.go) · [이벤트 캐시](../internal/serverapi/event_cache.go) · [UI 관찰](../internal/serverapi/ui_observer.go)

긴 원문은 조회 권한을 확인하고 정제한 뒤 UTF-8 바이트 페이지로 제공한다. 상세 캐시는 같은 세션·원본·리비전에서 정제한 원문을 재사용하며, 세션 종료 때 해제한다. 긴 도구 결과는 필드 단위 페이지로도 읽을 수 있다. [원문 조회 계약](webui-api.md#의미-단위-원문과-검색) · [상세 캐시](../internal/serverapi/detail/cache.go) · [입장 예산](../internal/serverapi/admission/budget.go)

## TUI 입력과 표시

입력기는 일정 간격으로 grapheme 경계의 기준 위치를 보관해 가까운 지점부터 탐색한다. 편집하면 영향을 받은 뒷부분의 위치를 폐기한다. Markdown 스트림은 완성 블록을 재사용하고 바뀔 수 있는 끝부분을 파싱한다. 이전 배치에 영향을 주는 문법이나 너비 변경은 필요한 배치를 다시 계산한다. [TUI 표시 계약](usage.md#도구-결과와-markdown) · [입력 경계](../internal/tui/input/boundaries.go) · [증분 Markdown](../internal/tui/markdown/stream.go)

뷰포트는 줄 위치를 간격을 두고 기록해 보이는 구간을 찾고 캐시 보유량에 맞춰 파생 배치를 회수한다. 원문과 행의 탐색 위치는 표시 캐시의 수명과 분리한다. [Markdown 줄 위치](../internal/tui/viewport/markdown_index.go) · [표시 캐시 예산](../internal/tui/viewport/cache_budget.go)

## WebUI 상태와 렌더링

서버 데이터는 TanStack Query에 캐시하고 초안·선택·진행 중 표시 상태는 Zustand에 둔다. 이벤트를 대상과 커서에 맞춰 적용하며, 저장 이력이 완료 경계에 도달할 때 실시간 본문을 교체한다. 숨겨진 채팅은 본문 갱신을 멈추고 승인·완료·예약을 계속 처리한다. 비활성 조회는 개수와 보유 바이트 기준으로 정리한다. [WebUI 상태 책임](webui-design.md#상태동시성저장-책임) · [이력 인계](../webui/src/pages/workspace/model/history-handoff.ts) · [표시 수명](../webui/src/pages/workspace/model/presentation.ts) · [조회 캐시](../webui/src/pages/workspace/api/cache-policy.ts)

대화와 파일 목록은 해당 화면 구간만 렌더링하며 포커스·드래그 중에 필요한 행을 보존한다. Markdown에는 별도 표시 예산을 둔다. 컴포저는 정상적인 텍스트 변경 범위의 UTF-8 바이트 차이를 적용하고, IME·붙여넣기처럼 범위를 확인할 수 없는 입력은 전체를 검산한다. [대화 가상화](../webui/src/pages/workspace/ui/chat/use-history-viewport.ts) · [파일 가상화](../webui/src/pages/workspace/ui/files/directory-column.tsx) · [Markdown 표시](../webui/src/shared/ui/markdown/render-budget.ts) · [입력 변경 범위](../webui/src/pages/workspace/model/use-input-edit.ts)

코드 편집기의 문법 강조는 별도의 처리 예산 안에서 CSS Highlight 범위로 그린다. 편집·미리보기 전환·닫기에서 이전 범위를 해제한다. 강조 예산을 넘는 문서는 일반 텍스트로 편집할 수 있다. [문법 강조](../webui/src/shared/ui/code-editor/syntax.ts) · [편집기 수명](../webui/src/shared/ui/code-editor/index.tsx)

## 빌드와 정적 자산

Vite 빌드는 React Compiler를 사용하고 초기 화면에 필요한 의존성과 지연 로드 화면을 분리한다. 자산 생성기는 내용이 같은 gzip 결과를 검증해 재사용한다. full 실행 파일에서 사전 압축한 자산을 제공하며 해시 경로에는 장기 캐시, 진입 HTML·favicon·빌드 식별자에는 재검증을 적용한다. [빌드 설정](../webui/vite.config.ts) · [압축 자산 생성](../devtools/cmd/uiassets/compression.go) · [자산 전송](../internal/webui/assets.go)

tui 빌드는 서버와 내장 WebUI를 의존성에서 제외한다. [배포 종류](releases.md#배포-파일) · [빌드 경계](development.md#배포-종류와-빌드-경계)

## 운영 로그

중앙 로거는 제한된 큐와 단일 writer를 사용한다. 경고 슬롯을 분리하고 같은 경고는 필드를 인코딩하기 전에 집계한다. 출력 포화는 집계로 알리며 종료 시 큐를 비우고 writer를 회수한다. 화면과 저장 이력의 사건 순서는 실행 경로가 소유한다. [로그 수명](diagnostics.md#수명과-보관) · [큐와 반복 경고](../internal/diagnostics/logger.go) · [writer 수명](../internal/diagnostics/writer.go)
