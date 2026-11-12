---
name: MECA Embedded WebUI
description: Astryx Neutral의 흑백 표면과 기능적 glass로 읽고 실행하는 Operate 작업대.
colors:
  accent: "light-dark(#191d24, #eef0f5)"
  on-accent: "light-dark(#ffffff, #191d24)"
  background-body: "light-dark(#f2f4f7, #0d0f13)"
  background-surface: "light-dark(#fcfdff, #121419)"
  background-card: "light-dark(#ffffff, #1a1d23)"
  background-popover: "light-dark(rgb(255 255 255 / 88%), rgb(26 29 35 / 92%))"
  background-muted: "light-dark(#edf0f4, #1e2128)"
  sidebar: "light-dark(rgb(255 255 255 / 78%), rgb(23 25 31 / 84%))"
  text-primary: "light-dark(#191d24, #eef0f5)"
  text-secondary: "light-dark(#616876, #adb3bf)"
  accent-muted: "light-dark(rgb(0 0 0 / 6%), rgb(255 255 255 / 10%))"
  neutral: "light-dark(#0000000F, #ffffff1A)"
  border: "light-dark(#00000014, #ffffff1A)"
  border-emphasized: "light-dark(#d4d4d4, #525252)"
  overlay-hover: "light-dark(#0000000D, #ffffff0D)"
  overlay-pressed: "light-dark(#0000001A, #ffffff1A)"
  success: "light-dark(#198100, #64af4c)"
  success-muted: "light-dark(#bce0bb, #90ca903D)"
  on-success-muted: "light-dark(#00490b, #90ca90)"
  warning: "#ffce2f"
  warning-muted: "light-dark(#fae19e, #e2b6233D)"
  on-warning-muted: "light-dark(#4b3900, #eec448)"
  error: "light-dark(#c9303a, #ff705d)"
  error-muted: "light-dark(#ffc4be, #ff98903D)"
  on-error-muted: "light-dark(#76000c, #ffaea7)"
typography:
  display-2: {"fontFamily": "system-ui, -apple-system, BlinkMacSystemFont, sans-serif", "fontSize": "2.1875rem", "fontWeight": 400, "lineHeight": 1.2571}
  heading-2: {"fontFamily": "system-ui, -apple-system, BlinkMacSystemFont, sans-serif", "fontSize": "1.25rem", "fontWeight": 600, "lineHeight": 1.4}
  body: {"fontFamily": "system-ui, -apple-system, BlinkMacSystemFont, sans-serif", "fontSize": "0.875rem", "fontWeight": 400, "lineHeight": 1.4286}
  label: {"fontFamily": "system-ui, -apple-system, BlinkMacSystemFont, sans-serif", "fontSize": "0.875rem", "fontWeight": 500, "lineHeight": 1.4286}
  supporting: {"fontFamily": "system-ui, -apple-system, BlinkMacSystemFont, sans-serif", "fontSize": "0.75rem", "fontWeight": 400, "lineHeight": 1.6667}
  code: {"fontFamily": "ui-monospace, \"SF Mono\", Monaco, Consolas, \"Liberation Mono\", \"Courier New\", monospace", "fontSize": "0.875rem", "fontWeight": 400, "lineHeight": 1.4286}
rounded:
  inner: "0.375rem"
  element: "0.625rem"
  container: "0.75rem"
  page: "1.75rem"
  chat: "24px"
  full: "9999px"
spacing:
  "1": "0.25rem"
  "2": "0.5rem"
  "3": "0.75rem"
  "4": "1rem"
  "5": "1.25rem"
  "6": "1.5rem"
  "8": "2rem"
components:
  button-primary: {"backgroundColor": "{colors.accent}", "textColor": "{colors.on-accent}", "typography": "{typography.label}", "rounded": "{rounded.element}", "padding": "0.5rem 0.75rem", "height": "32px"}
  button-secondary: {"backgroundColor": "{colors.neutral}", "textColor": "{colors.text-primary}", "typography": "{typography.label}", "rounded": "{rounded.element}", "padding": "0.5rem 0.75rem", "height": "32px"}
  button-ghost: {"backgroundColor": "transparent", "textColor": "{colors.text-primary}", "typography": "{typography.label}", "rounded": "{rounded.element}", "padding": "0.5rem 0.75rem", "height": "32px"}
  button-destructive: {"backgroundColor": "{colors.error-muted}", "textColor": "{colors.on-error-muted}", "typography": "{typography.label}", "rounded": "{rounded.element}", "padding": "0.5rem 0.75rem", "height": "32px"}
  input: {"backgroundColor": "{colors.background-surface}", "textColor": "{colors.text-primary}", "typography": "{typography.body}", "rounded": "{rounded.element}", "padding": "0.25rem 0.5rem"}
  conversation-nav: {"backgroundColor": "{colors.neutral}", "textColor": "{colors.text-primary}", "typography": "{typography.label}", "rounded": "{rounded.element}", "padding": "0 0.5rem", "height": "32px"}
  reference-token: {"backgroundColor": "{colors.neutral}", "textColor": "{colors.text-primary}", "typography": "{typography.supporting}", "rounded": "{rounded.inner}", "padding": "0 0.5rem", "height": "24px"}
  message-filled: {"backgroundColor": "{colors.neutral}", "textColor": "{colors.text-primary}", "typography": "{typography.body}", "rounded": "{rounded.chat}", "padding": "0.75rem 1rem"}
  composer: {"backgroundColor": "{colors.background-popover}", "textColor": "{colors.text-primary}", "typography": "{typography.body}", "rounded": "{rounded.chat}", "padding": "0.75rem"}
  thinking: {"backgroundColor": "light-dark(rgb(235 238 242 / 70%), rgb(31 34 41 / 84%))", "textColor": "light-dark(#606774, #b4bac5)", "typography": "{typography.body}"}
---

# Design System: MECA Embedded WebUI

## Overview

**디자인 방향: 읽기와 실행에 집중하는 작업대**

흑백 중심의 표면과 얇은 glass로 읽기와 실행에 집중하는 작업대다. 프로젝트에서 대화를 고르고, 입력한 뒤 답변·Thinking·도구 진행을 읽는 흐름이 화면의 중심이다. 넓은 읽기 영역과 밀도 있는 탐색·설정의 차이로 위계를 만들며, 색은 의미가 있는 상태에 사용한다.

Astryx 공식 Neutral을 확장한 현재 구현을 기록한다. 제품 계약은 [SYSTEM.md](../SYSTEM.md), 화면 계약은 [내장 WebUI 구조와 디자인](../docs/webui-design.md)이 소유한다. 시각 값과 공식 컴포넌트 확장은 [theme.ts](src/app/styles/theme.ts), 스타일 조합은 [base.css](src/app/styles/base.css), 테마 적용은 [main.tsx](src/app/main.tsx)가 소유한다. 공식 ai-chat·ai-chat-landing의 구성 언어를 따르며 장식 이미지·외부 글꼴을 추가하지 않는다. 브랜드 아이콘은 흑백 ME/CA 기하학 모노그램을 사용한다. `public/favicon.svg`의 단일 벡터 도형을 브라우저 파비콘과 사이드바의 Astryx `Icon`이 공유한다. 로그인·시작 화면은 파비콘 없이 `shared/ui/wordmark`의 가로형 MECA 글자를 사용하며 기하학 도형과 접근성 제목을 공유하고 현재 본문색을 따른다.

**Key Characteristics:**

- 기본 라이트와 저장된 라이트·다크·시스템 선택.
- 흑백 동작·중성 선택과 Astryx의 의미별 상태색.
- 운영체제 한글 글꼴, 읽기와 조작의 다른 밀도.
- 겹친 표면의 기능적 glass와 공식 모션·reduced-motion.
- Astryx가 소유하는 위젯·키보드·포커스·모달 수명.

## Colors

frontmatter는 실제 생성 CSS의 `light-dark()` 표기를 유지한다. 첫 값이 라이트, 둘째가 다크다. `accent`·표면·본문·보조 잉크는 MECA 확장이고, 상태·경계·중성 조작은 공식 Neutral에서 상속한다. CSS 사용자 정의 속성 이름은 같은 이름 앞에 `--color-`를 붙이며 `sidebar`만 SideNav의 공식 컴포넌트 확장 값이다. 실제 색상 값은 `src/app/styles/theme.ts`와 상속한 Neutral 테마가 소유한다.

### Primary

- **단색 동작 / 동작 위 글자** (`accent`, `on-accent`): 주요 버튼·전송과 그 위 글자·아이콘. 포커스 윤곽도 같은 accent를 따른다.

### Secondary

- **연결·완료** (`success`): 연결 점·성공 상태. 의미 배경과 글자는 공식 `success-muted`·`on-success-muted`를 사용한다.
- **승인·중단** (`warning`): 주의 점·경고 상태와 해당 의미 배경·글자 쌍.
- **실패** (`error`): 실패 점·오류 상태. 파괴적 버튼은 공식 Neutral의 muted 오류 배경·글자 쌍이다.

### Neutral

- **바깥 바탕 / 읽기 표면 / 카드 / 보조면** (`background-body`, `background-surface`, `background-card`, `background-muted`): 화면과 콘텐츠 층위를 조용히 구분한다.
- **떠 있는 표면 / 탐색 재질** (`background-popover`, `sidebar`): 겹침에 쓰는 반투명 표면.
- **본문 / 보조 잉크** (`text-primary`, `text-secondary`): 답변과 메타데이터·Thinking의 위계.
- **중성 조작 / 경계 / 강조 경계** (`neutral`, `border`, `border-emphasized`): 보조 버튼·선택 행·폼 경계. Hover·press는 공식 overlay를 따른다.

상태는 공식 의미 토큰과 글자·아이콘·접근성 상태를 함께 사용한다. 주요 동작과 팀 선택은 중성 체계를 유지한다.

코드의 문법 의미색은 Core·Neutral의 공식 syntax 토큰을 상속한다. 이를 팀색이나 chrome 장식색으로 사용하지 않는다.

## Typography

**Display Font:** 시스템 heading 글꼴. 빈 대화의 한 제목에 공식 `display-2`를 사용한다.
**Body Font:** `system-ui`와 운영체제 fallback. 한글도 운영체제 글꼴을 사용하며 외부 폰트를 로드하지 않는다.
**Label/Mono Font:** UI 레이블은 같은 sans-serif, 코드·경로·정량 정보는 공식 code 글꼴이다.

### Hierarchy

- **Display** (`display-2`): 빈 대화의 질문 제목. 과업용 UI 제목이며 별도 장식용 글꼴을 도입하지 않는다.
- **Title** (`heading-2`): Dialog·Drawer 등 표면의 제목.
- **Body** (`body`): 답변·입력·폼 본문. Markdown의 제목·목록·코드 위계는 공식 컴포넌트가 소유한다.
- **Label** (`label`): 버튼·탐색·폼 이름. 필요 시 공식 `weight`로 강조한다.
- **Supporting** (`supporting`): 발신자·Thinking 트리거·연결 복구 안내. 완료 측정과 컴포저 사용량은 `size="base"`인 14px 캡션과 `hasTabularNumbers`를 사용한다. 횟수는 세 자리씩 묶고 이번 응답과 대화 누적값의 범위를 툴팁·접근성 이름으로 구별한다.
- **Code** (`code`): 공식 CodeBlock·기본 텍스트 편집기와 경로·수치.

크기는 rem 기반의 실제 기본 역할 값이다. Astryx의 dynamic type·입력 환경 적응을 유지하며, Markdown과 에디터 전체에 단일 글자 크기를 덮어쓰지 않는다.

긴 답변과 조작 영역의 밀도를 구분한다. Thinking의 본문과 제목은 같은 secondary 의미색으로 읽히게 한다.

## Layout

AppShell의 surface 구성 안에서 SideNav와 유동 Layout을 결합한다. 상단에는 대화·파일·스킬 TabList와 접힌 사이드바를 다시 여는 버튼을 배치한다. 긴 이름이 탭을 밀어내지 않도록 하고 전체 경로·제목은 툴팁으로 읽는다. 설정과 빠른 테마 전환은 탐색 하단에 둔다. 작업 흐름은 프로젝트 → 대화 → 입력 → 답변·Thinking·도구 → 완료·승인이다.

대화와 컴포저는 각 채팅 컨테이너 폭의 80%를 사용한다. 대화의 좌우 여백은 ChatMessageList가 소유하며 바깥 래퍼에 중복 적용하지 않는다. 세로 여백은 `spacing.4`, 메시지 간 간격은 `spacing.5`다. 컴포저 바깥은 `spacing.4`, 내부는 `spacing.3`을 사용한다. 비어 있는 대화는 한 제목·보조 설명과 컴포저를 함께 읽히게 배치하고, 이력이 있으면 ChatMessageList 안에 가상 이력과 live 출력을 함께 두고 하단에 컴포저를 배치한다. 새 요청이 시작되면 대화 하단으로 이동하고 이후의 높이 변화와 위로 스크롤한 뒤의 읽기 위치는 ChatLayout이 관리한다.

- **탐색 반응형:** AppShell의 기본 md 경계 (768px 미만)에서 SideNav가 모바일 drawer로 전환한다. 긴 프로젝트·대화 이름은 행 안에서 축약하고 선택·작업 상태를 유지한다.
- **설정 반응형:** Dialog 요청 폭은 (1040px), 넓은 화면의 탐색은 (224px)이다. (900px) 이하에서는 상단 Selector, (480px) 이하의 테마 선택도 Selector를 사용한다. 탐색 레이블은 글자 확대 시 줄바꿈하며 같은 폼을 유지해 편집 중인 값을 보존한다.
- **보조 작업:** 파일은 넓은 창에서 가운데 문서·오른쪽 현재 폴더 목록 하나를 두고 왼쪽 프로젝트 탐색을 유지한다. 목록 상단은 작업공간 텍스트·우측 끝 새로고침이며 폴더 이동도 목록 안에 둔다. 에디터 위의 별도 탐색 줄은 없다. 선택 즉시 Raw 편집기를 열며 Preview/Raw로 현재 초안과 원문을 전환한다. 미리보기 본문 클릭은 Raw 편집으로 돌아가되 링크·복사·텍스트 선택을 방해하지 않는다. 편집기 수명·실행 취소 이력을 보존하며 미리보기 코드 블록은 긴 단어도 줄바꿈한다. 스킬은 넓은 창에서 목록·문서, 파일·스킬 모두 (1024px) 이하에서 목록·본문 전환을 사용한다. 원문 Drawer의 요청 폭은 (760px)이며 실제 화면 가용 폭은 공식 컴포넌트가 제한한다.

간격은 공식 rem 기반 4px 리듬을 사용한다. 이 문서에 추출한 단계는 실제 앱에서 반복되는 부분집합이다. 화면을 줄일 때 조작 의미·포커스·정보 순서를 유지한다.

## Elevation & Depth

읽기·파일 영역은 불투명 중성 표면을 사용한다. Glass는 SideNav·고정 헤더·컴포저·Popover·DropdownMenu처럼 겹침의 관계를 설명하는 비모달 표면에만 적용한다. blur는 18px이며 얇은 안쪽 빛과 절제된 그림자로 경계를 구분한다. Dialog·AlertDialog·Drawer·Lightbox의 본문 표면은 항상 불투명하며 backdrop blur를 적용하지 않는다. Dialog 진입의 opacity 애니메이션도 사용하지 않는다. 배경을 어둡게 하는 scrim과 모달 본문의 불투명도는 구분한다.

### Shadow Vocabulary

- **Low** (`--shadow-low`): 컴포저 등 기본 떠 있는 표면의 낮은 층위.
- **Medium** (`--shadow-med`): 컴포저 hover·focus와 더 높은 표면의 강조.
- **High** (`--shadow-high`): Dialog 등 공식 overlay의 더 깊은 층위. Neutral의 상속 값이다.

라이트는 차가운 중성 바탕과 명료한 경계, 다크는 어두운 중성 바탕과 얇은 반사를 사용한다. 흑백 중심의 동작·선택과 의미별 상태색으로 위계를 구분한다. blur 미지원·reduced-transparency·forced-colors에서는 불투명 표면으로 전환한다.

모션은 공식 테마의 fast·medium·slow와 변동 비율, overshoot 없는 crisp 성향을 사용한다. 버튼·탭·Switch·Tooltip·Drawer·Skeleton·Spinner의 내재 모션은 Astryx가 소유한다. 추가한 상태 모션은 `+`의 회전(90deg), 작업 탭의 8px slide-in, Dialog의 12px slide-in, 상세 본문의 4px 진입이다. 진입은 ease-entrance의 감속 곡선을 사용하며 Dialog는 220ms, 탭·상세는 fast-max로 동작한다. 모든 모달과 툴 본문은 애니메이션 중에도 불투명도를 유지한다. 예약 목록의 조작을 가리는 클리핑 효과는 사용하지 않는다. 프로젝트 대화 목록은 Astryx useEntryAnimation의 slideDown, 메뉴의 진입·닫기는 공식 DropdownMenu가 소유한다. 응답 토큰마다 진입을 재시작하거나 임의의 목록 stagger를 추가하지 않는다. `prefers-reduced-motion`에서는 공식 적응을 유지하고 추가한 회전·펼침의 시간도 즉시 전환한다.

## Shapes

`inner`는 참조 Token, `element`는 버튼·입력·탐색 행, `container`는 작은 표면, `page`는 공식 큰 표면, `full`은 원형·pill 조작이다. 대화 bubble과 컴포저는 확장한 `chat` 반경을 사용한다. 컴포저 안의 버튼은 공식 동심 반경 계산을 따른다. 경계는 공식 얇은 선 (1px)을 사용하며 모든 표면을 하나의 반경으로 맞추지 않는다.

## Components

### Buttons

조용한 중성 조작과 분명한 주요 동작을 구분한다. `Button`의 primary·secondary·ghost·destructive와 `IconButton`을 사용한다. 모서리·여백·크기는 공식 토큰을 따르며 hover·press·loading·disabled를 보존한다. 키보드 포커스는 공식 윤곽 (2px)과 바깥 간격 (3px)을 사용한다. 전송은 별도 수제 버튼 대신 ChatSendButton이 소유한다.

### Chips

파일·폴더·스킬 참조는 아이콘을 포함한 ChatComposerTokenElement·ChatTokenizedText로 입력과 전송한 메시지에 표시한다. 긴 경로는 줄바꿈하고 툴팁으로 전체 경로를 제공한다. 참조를 추가하면 현재 파일·스킬 보기를 유지하고 메시지를 전송한 뒤 대화로 이동한다. 첨부 이미지는 컴포저 위에 독립된 작은 glass 아일랜드로 표시한다. Astryx Carousel의 한 줄 가로 탐색과 64px Thumbnail을 사용하며 우측 상단 원형 X는 hover·키보드 포커스에서 나타나고 터치 환경에서는 항상 보인다. 파일명은 Thumbnail의 접근성 이름과 툴팁으로 전달한다. 명령·스킬은 ChatComposerInput의 typeahead로 선택한다. 저장된 이미지는 Thumbnail에서 Lightbox로 연다. 상태 Badge·StatusDot는 의미 토큰과 접근성 레이블을 함께 사용하며 팀은 ToggleButton의 눌림 상태로 표현한다.

### Cards / Containers

사용자 메시지는 채워진 ChatMessageBubble, 답변은 ghost bubble과 Markdown으로 구분한다. 답변·Thinking·ChatToolCalls는 할당된 채팅 열의 전체 너비를 유지하며 내용 길이에 따라 폭을 바꾸지 않는다. Markdown·코드·표는 영역 안에서 줄바꿈해 가로 스크롤을 만들지 않는다. 완료 정보는 본문 없는 RunCompletion API와 전용 CompletionRow를 사용해 마지막 출력 뒤 ChatSystemMessage로 표시한다. 토큰·TPS·시간·비용을 표시하고 실패·중단은 고정된 UI 문구로 구분한다. 응답·추론·자유 형식 종료 사유나 성공 완료 라벨·녹색 점은 넣지 않는다. 실패 원문은 별도 상세 창으로 연다. 팀 기여는 작성자·요약·경로·계약을 가진 본문으로 표시한다. 재시도·요약·스킬 읽기 알림은 고정 코드에서 만든 왼쪽 정렬 보조 문구이며 완료 행과 표시 컴포넌트를 공유하지 않는다. 실시간 출력은 해당 작업의 저장 경계와 조회 revision을 확인한 뒤 이력으로 넘긴다. 대화 전체를 장식 카드로 감싸지 않는다. 작업·원문은 Drawer, 승인·설정은 Dialog, 편집 손실·설정 저장 확인은 AlertDialog가 표시·포커스·Escape 수명을 소유한다.

### Tool disclosures and long responses

툴콜은 배경·외곽선·카드 없는 Astryx Collapsible이며 툴 이름 밑줄과 상태 아이콘만 표시한다. 같은 행의 별도 새 창 모양 아이콘은 요청 인자만 읽기 좋은 레이블·코드로 보여주는 불투명 Dialog를 연다. 결과는 아코디언 안에서만 펼치며 stdout·본문·diff를 먼저, 실행 식별자 등 기술 메타데이터를 접힌 보조 영역에 표시한다. 일반 결과 표면에는 glass를 사용하지 않는다.

답변·Thinking에 전체 텍스트 팝오버나 원문 보기 버튼을 붙이지 않는다. 표시 접두부보다 긴 저장 본문은 같은 위치에서 원문 범위별로 읽는다. 노드가 많은 목록·표는 별도 본문 페이지로 읽으며 끝부분을 삭제하지 않는다. 툴 결과와 요청은 펼치거나 요청 아이콘을 선택할 때만 조회하고 닫으면 브라우저 페이지 캐시를 해제한다. 가상화한 저장 행은 일반 문서 흐름에 배치해 비동기 결과가 커지는 즉시 다음 행이 아래로 밀려나게 한다. 위·아래 spacer와 제한된 마운트 수로 긴 이력의 DOM을 제한한다.

### Inputs / Fields

설정 폼은 FormLayout·TextInput·Selector·Switch·CheckboxInput을 사용한다. 레이블·설명·오류·저장 상태를 같은 폼 문맥에서 읽히게 한다. 저장 버튼은 확인 모달을 열고, 네를 선택해 저장이 성공하면 설정 창을 닫는다. 실패하면 편집 값과 오류를 유지한다. 필드는 중성 표면·강조 경계에서 accent 포커스로 전환하며 공식 inset 피드백과 disabled 처리를 유지한다. 모바일 글자 크기·키보드 환경 적응도 공식 컴포넌트가 소유한다. 편집기는 react-simple-code-editor와 Core의 공개 tokenizer·syntax 토큰을 사용하는 예외이며, 나머지 UI는 Astryx 컴포넌트를 사용한다.

### Navigation

프로젝트는 접는 SideNavItem 그룹, 대화는 들여쓴 선택 행이다. 사이드바 접기·펼치기는 레일 상단에, 설정·테마는 하단에 둔다. Toolbar의 TabList는 작업 보기, 끝의 DropdownMenu는 언어를 선택하고 현재 언어를 체크 표시로 구별한다. 테마는 하단 IconButton으로 전환한다. 모바일 탐색은 AppShell이 소유한다. 공용 컨트롤은 Astryx Neutral의 Lucide registry를 상속하며 파일 형식 glyph만 공통 32종 카탈로그의 MUI Material 아이콘을 사용한다.

디렉터리 경로는 Astryx Breadcrumbs로 위치 행의 80%를 사용한다. 마지막 세 단계까지 표시하고 너비가 모자라면 앞쪽부터 … 메뉴로 접는다. 모달 본문은 불투명을 유지하며 테마 팔레트는 배경과 글자색을 함께 즉시 바꿔 중간 회색에서 대비가 낮아지는 현상을 막는다. 사이드바의 너비 애니메이션은 Astryx가 계속 소유하며 모션 감소 설정에서는 추가 전환을 생략한다.

### Composer and Thinking

전송 버튼의 툴팁은 전송(대기 중)·방향 수정(응답 중)·후속 예약(응답 중)을 각각 한 행에 표시한다. 키 표시는 Astryx Kbd가 소유하며 Enter와 `mod+enter`를 사용해 플랫폼에 맞는 Control·Command 키를 보여준다. 키보드 입력 처리는 컴포저가 계속 소유한다. 키캡의 잉크·바탕·밑선은 툴팁의 전경색을 기준으로 파생해 두 테마에서 대비를 유지한다.

접속 확인에서 작업 화면으로 넘어가는 시작 상태는 단일 glass 표면과 Spinner로 표현한다. MECA 이름과 실제 연결/작업 공간 준비 안내를 보여주고 진행률·장식용 스켈레톤·최소 대기 시간을 추가하지 않는다. 시작 화면의 재질은 공통 glass 토큰을 사용하며 모달의 불투명 규칙과 구분한다.

컴포저는 입력·참조 Token·팀 토글·후속 예약·전송을 묶는 glass 작업 표면이다. 모델은 설정과 `!model`에서 선택한다. 음성·dictation 조작은 없다. `+`는 위쪽 DropdownMenu에서 이미지와 스킬을 선택한다. 이미지는 파일 선택창을, 스킬은 별도 탐색 모달을 열며 행 우측 +로 입력에 넣는다. 메뉴·모달의 Escape·선택 완료와 포커스 복귀는 공식 레이어 동작을 사용한다. 컴포저 전체의 이미지 드롭·붙여넣기·명시 단축키는 같은 첨부 한도를 따른다. 후보는 `!` 명령·`#` 스킬의 공식 typeahead다.

Thinking은 기본 펼침 Collapsible과 스트리밍 Markdown의 조합이다. 차분한 중성 배경·container 반경·spacing.4 안쪽 여백을 사용하고 답변 앞에는 spacing.6(24px)을 둔다. `data-tone="thinking"`의 테마 확장으로 라이트 #606774, 다크 #b4bac5의 보조 잉크를 적용한다. 각 저장 응답 그룹 뒤에는 spacing.8(32px)을 둔다. 진행 중과 저장 이력에서 같은 표현을 유지하며 답변은 별도 기본 본문색과 1.7 행간으로 읽는다. 아코디언 트리거는 데스크톱에서 최소 32px, coarse pointer에서 최소 48px의 조작 높이를 확보한다.

컴포저 하단 한 줄에는 메모리·입력/출력 토큰·소요시간·TPS 등 실행 지표를 표시한다. 대기·실행·압축 같은 상태 문구나 SSE 조작은 배치하지 않으며 좁은 화면에서는 지표를 줄바꿈한다.

## 적용 원칙

- 두 테마는 공통 의미 토큰으로 연결하며 상태의 글자·아이콘·바탕과 대비를 함께 확인한다.
- 조작과 포커스·선택·모달 수명은 Astryx 컴포넌트가 소유한다. 코드 편집기에는 명시한 예외 계약을 적용한다.
- 긴 탐색 레이블은 행 안에서 줄이고 선택 항목·핵심 조작과 전체 이름의 툴팁을 제공한다.
- 스트리밍과 저장 이력은 답변·Thinking·도구의 같은 정보 위계를 사용한다.
- 좁은 설정은 상단 Selector로 탐색하고 폼에 필요한 폭을 확보한다.
- 모델 선택은 설정과 `!model`, 이미지·스킬 추가는 컴포저의 `+`에서 제공한다. 입력 명령은 `!`로 시작한다.
- 상태는 색과 글자·모양을 함께 사용하며 축소 모션에서는 추가 공간 전환을 생략한다.
- 읽기와 입력이 대화 화면의 중심이다. 제품 문구와 상태는 실제 사용 흐름과 서버 결과를 따른다.
