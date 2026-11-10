---
name: MECA Embedded WebUI
description: Astryx Neutral의 흑백 표면과 기능적 glass로 읽고 실행하는 Operate 작업대.
colors:
  accent: "light-dark(#171717, #f5f5f5)"
  on-accent: "light-dark(#ffffff, #171717)"
  background-body: "light-dark(#f5f5f5, #0b0b0b)"
  background-surface: "light-dark(#ffffff, #141414)"
  background-card: "light-dark(#ffffff, #191919)"
  background-popover: "light-dark(rgb(255 255 255 / 88%), rgb(28 28 28 / 92%))"
  background-muted: "light-dark(#f5f5f5, #202020)"
  sidebar: "light-dark(rgb(250 250 250 / 82%), rgb(18 18 18 / 86%))"
  text-primary: "light-dark(#171717, #f5f5f5)"
  text-secondary: "light-dark(#666666, #aaaaaa)"
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
  thinking: {"backgroundColor": "transparent", "textColor": "{colors.text-secondary}", "typography": "{typography.body}"}
---

# Design System: MECA Embedded WebUI

## Overview

**디자인 방향: 읽기와 실행에 집중하는 작업대**

흑백 중심의 표면과 얇은 glass로 읽기와 실행에 집중하는 작업대다. 프로젝트에서 대화를 고르고, 입력한 뒤 답변·Thinking·도구 진행을 읽는 흐름이 화면의 중심이다. 넓은 읽기 영역과 밀도 있는 탐색·설정의 차이로 위계를 만들며, 색은 의미가 있는 상태에 사용한다.

Astryx 공식 Neutral을 확장한 현재 구현을 기록한다. 제품 계약은 [SYSTEM.md](../SYSTEM.md), 화면 계약은 [내장 WebUI 구조와 디자인](../docs/webui-design.md)이 소유한다. 시각 값과 공식 컴포넌트 확장은 [theme.ts](src/app/styles/theme.ts), 스타일 조합은 [base.css](src/app/styles/base.css), 테마 적용은 [main.tsx](src/app/main.tsx)가 소유한다. 공식 ai-chat·ai-chat-landing의 구성 언어를 따르며 새로운 브랜드·장식 이미지·외부 글꼴을 추가하지 않는다.

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

코드의 문법 의미색은 Neutral·Lab의 공식 syntax 토큰을 상속한다. 이를 팀색이나 chrome 장식색으로 사용하지 않는다.

## Typography

**Display Font:** 시스템 heading 글꼴. 빈 대화의 한 제목에 공식 `display-2`를 사용한다.
**Body Font:** `system-ui`와 운영체제 fallback. 한글도 운영체제 글꼴을 사용하며 외부 폰트를 로드하지 않는다.
**Label/Mono Font:** UI 레이블은 같은 sans-serif, 코드·경로·정량 정보는 공식 code 글꼴이다.

### Hierarchy

- **Display** (`display-2`): 빈 대화의 질문 제목. 과업용 UI 제목이며 별도 장식용 글꼴을 도입하지 않는다.
- **Title** (`heading-2`): Dialog·Drawer 등 표면의 제목.
- **Body** (`body`): 답변·입력·폼 본문. Markdown의 제목·목록·코드 위계는 공식 컴포넌트가 소유한다.
- **Label** (`label`): 버튼·탐색·폼 이름. 필요 시 공식 `weight`로 강조한다.
- **Supporting** (`supporting`): 발신자·완료 정보·Thinking 트리거·연결 복구 안내.
- **Code** (`code`): 공식 CodeBlock·Lab CodeEditor와 경로·수치.

크기는 rem 기반의 실제 기본 역할 값이다. Astryx의 dynamic type·입력 환경 적응을 유지하며, Markdown과 에디터 전체에 단일 글자 크기를 덮어쓰지 않는다.

긴 답변과 조작 영역의 밀도를 구분한다. Thinking의 본문과 제목은 같은 secondary 의미색으로 읽히게 한다.

## Layout

AppShell의 surface 구성 안에서 SideNav와 유동 Layout을 결합한다. 상단 Toolbar는 프로젝트·대화 문맥, 가운데 TabList는 대화·파일·스킬, 우측은 작업 목록을 제공한다. 설정과 빠른 테마 전환은 탐색 하단에 둔다. 작업 흐름은 프로젝트 → 대화 → 입력 → 답변·Thinking·도구 → 완료·승인이다.

대화와 컴포저는 같은 최대 폭 (760px)을 사용한다. 대화의 안쪽 여백은 `spacing.6`, 세로 여백은 `spacing.4`, 메시지 간 간격은 `spacing.5`다. 컴포저 바깥은 `spacing.4`, 내부는 `spacing.3`을 사용한다. 비어 있는 대화는 한 제목·보조 설명과 컴포저를 함께 읽히게 배치하고, 이력이 있으면 ChatMessageList 안에 가상 이력과 live 출력을 함께 두고 하단에 컴포저를 배치한다. 스크롤·맨 아래 이동은 ChatLayout이 소유한다.

- **탐색 반응형:** AppShell의 기본 md 경계 (768px 미만)에서 SideNav가 모바일 drawer로 전환한다. 긴 프로젝트·대화 이름은 행 안에서 축약하고 선택·작업 상태를 유지한다.
- **설정 반응형:** Dialog 요청 폭은 (840px), 넓은 화면의 탐색은 (168px)이다. (640px) 이하에서는 고정 탐색 대신 상단 Selector를 사용한다. 같은 폼을 유지해 편집 중인 값과 폼 폭을 보존한다.
- **보조 작업:** 파일·스킬은 Layout의 컬럼·문서 위계를 사용한다. 작업 Drawer 요청 폭은 (480px), 원문 Drawer는 (760px)이며 실제 화면 가용 폭은 공식 컴포넌트가 제한한다.

간격은 공식 rem 기반 4px 리듬을 사용한다. 이 문서에 추출한 단계는 실제 앱에서 반복되는 부분집합이다. 화면을 줄일 때 조작 의미·포커스·정보 순서를 유지한다.

## Elevation & Depth

읽기·파일 영역은 불투명 중성 표면과 얇은 경계로 구분한다. 겹친 Dialog·Popover·컴포저에 기능적 glass를 사용한다. Dialog blur는 (24px), DropdownMenu·Popover·컴포저 blur는 (20px)이며, 불투명 fallback 등 환경 적응은 공식 Astryx 동작을 유지한다. 그림자는 Neutral의 공식 토큰을 사용한다.

### Shadow Vocabulary

- **Low** (`--shadow-low`): 컴포저 등 기본 떠 있는 표면의 낮은 층위.
- **Medium** (`--shadow-med`): 컴포저 hover·focus와 더 높은 표면의 강조.
- **High** (`--shadow-high`): Dialog 등 공식 overlay의 더 깊은 층위. Neutral의 상속 값이다.

Glass는 겹친 표면의 층위를 설명한다. 농도와 blur를 함께 유지하고 읽기 본문·편집기까지 확장하지 않는다.

모션은 공식 테마의 fast·medium·slow와 변동 비율을 사용한다. 버튼·탭·Switch·Tooltip·Drawer·Skeleton·Spinner의 내재 모션은 Astryx가 소유한다. 추가한 시그니처는 `+`의 회전 (90deg)과 위로 펼치는 ChatComposerDrawer이며 medium 시간·standard easing으로 함께 움직인다. 응답 토큰마다 진입을 재시작하거나 임의의 목록 stagger를 추가하지 않는다. `prefers-reduced-motion`에서는 공식 적응을 유지하고 추가한 회전·펼침의 시간도 즉시 전환한다.

## Shapes

`inner`는 참조 Token, `element`는 버튼·입력·탐색 행, `container`는 작은 표면, `page`는 공식 큰 표면, `full`은 원형·pill 조작이다. 대화 bubble과 컴포저는 확장한 `chat` 반경을 사용한다. 컴포저 안의 버튼은 공식 동심 반경 계산을 따른다. 경계는 공식 얇은 선 (1px)을 사용하며 모든 표면을 하나의 반경으로 맞추지 않는다.

## Components

### Buttons

조용한 중성 조작과 분명한 주요 동작을 구분한다. `Button`의 primary·secondary·ghost·destructive와 `IconButton`을 사용한다. 모서리·여백·크기는 공식 토큰을 따르며 hover·press·loading·disabled를 보존한다. 키보드 포커스는 공식 윤곽 (2px)과 바깥 간격 (3px)을 사용한다. 전송은 별도 수제 버튼 대신 ChatSendButton이 소유한다.

### Chips

입력에 보관한 파일과 이미지 이름은 제거 가능한 Token이다. 긴 이름은 토큰 안에서 축약하고 여러 토큰은 줄바꿈한다. 명령·스킬은 ChatComposerInput의 typeahead와 TokenElement로 표현한다. 저장된 이미지는 Thumbnail에서 Lightbox로 연다. 상태 Badge·StatusDot는 의미 토큰과 접근성 레이블을 함께 사용하며 팀은 ToggleButton의 눌림 상태로 표현한다.

### Cards / Containers

사용자 메시지는 채워진 ChatMessageBubble, 답변은 ghost bubble과 Markdown으로 구분한다. 도구는 ChatToolCalls, 완료 정보는 ChatSystemMessage·ChatMessageMetadata를 사용한다. 대화 전체를 장식 카드로 감싸지 않는다. 작업·원문은 Drawer, 승인·설정은 Dialog, 편집 손실 확인은 AlertDialog가 표시·포커스·Escape 수명을 소유한다.

### Inputs / Fields

설정 폼은 FormLayout·TextInput·Selector·Switch·CheckboxInput을 사용한다. 레이블·설명·오류·저장 상태를 같은 폼 문맥에서 읽히게 한다. 필드는 중성 표면·강조 경계에서 accent 포커스로 전환하며 공식 inset 피드백과 disabled 처리를 유지한다. 모바일 글자 크기·키보드 환경 적응도 공식 컴포넌트가 소유한다. 편집기는 문법 강조·줄 번호를 제공하는 Astryx Lab CodeEditor다.

### Navigation

프로젝트는 접는 SideNavItem 그룹, 대화는 들여쓴 선택 행이다. 실제 overflow 경계를 유지해 긴 이름이 사이드바 폭을 밀어내지 않게 한다. Toolbar의 TabList는 작업 보기를, 설정의 List·Selector는 설정 위치를, SegmentedControl은 테마 선택을 표현한다. 모바일 탐색은 AppShell이 소유하고 조작 아이콘은 Astryx Icon으로 표현한다. 공용 컨트롤은 Astryx Neutral의 기본 Lucide registry를 상속하고 앱 전용 아이콘도 같은 Lucide 라이브러리로 추가한다.

### Composer and Thinking

컴포저는 입력·참조 Token·팀 토글·후속 예약·전송을 묶는 glass 작업 표면이다. 모델은 설정과 `!model`에서 선택한다. 음성·dictation 조작은 없다. `+`를 누르면 이미지 전용 FileInput과 스킬 탐색 버튼이 입력 위로 펼쳐진다. 닫힌 패널은 inert이며 Escape·선택 완료로 닫고 입력에 포커스를 돌린다. 컴포저 전체의 이미지 드롭·붙여넣기·명시 단축키는 같은 첨부 한도를 따른다. 후보는 `!` 명령·`#` 스킬의 공식 typeahead다.

Thinking은 기본 펼침 Collapsible과 스트리밍 Markdown의 조합이다. `data-tone="thinking"`의 공식 테마 확장으로 primary 텍스트를 secondary로 바꾼다. 진행 중과 저장 이력에서 같은 표현을 사용하며 제목·목록까지 회색 계층을 유지한다.

하단에는 SSE 연결 상태를 한 곳에 둔다. 정상 연결은 녹색 점, 최초 연결·복구·일시정지는 각 레이블로 구분한다. 실행 지표는 열었을 때만 Popover에 표시한다.

## Do's and Don'ts

### Do:

- **Do** 의미 토큰으로 두 테마를 연결하고 상태의 글자·아이콘·바탕을 함께 확인한다.
- **Do** Astryx의 실제 포커스·선택·모달·reduced-motion 동작을 유지한다.
- **Do** 긴 탐색 레이블을 행 안에서 줄이고 선택 항목과 실제 조작을 보이게 한다.
- **Do** 스트리밍과 저장 이력에서 답변·Thinking·도구의 같은 위계를 유지한다.
- **Do** 좁은 설정에서는 상단 Selector와 폼의 사용 가능한 폭을 함께 확인한다.

### Don't:

- **Don't** 녹색 주요 버튼·보라색 팀 전용 팔레트·임의 회색 단계를 추가한다.
- **Don't** 위젯 내부 DOM이나 직접 만든 textarea·select·checkbox·modal CSS로 공식 조작을 대체한다.
- **Don't** 컴포저에 모델 선택·음성 입력을 되돌리거나 !명령을 slash 명령으로 바꾼다.
- **Don't** 색만으로 상태를 전달하거나 reduced-motion에서 추가한 회전·펼침 전환을 유지한다.
- **Don't** 대화 작업 영역을 장식 카드·지표 대시보드로 채우거나 fixture 문구를 제품 문구의 기준으로 취급한다.
