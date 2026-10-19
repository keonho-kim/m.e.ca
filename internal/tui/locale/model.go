// Purpose: Declare bilingual model identity and capability information labels.
// Pattern: Declarative presentation data.
// Usage: Model information projects verified provider facts beside localized labels.
// Related: internal/tui/locale/locale.go, internal/tui/modelmenu/identity.go.
package locale

var model = []translation{
	{"", "No model connected.", "연결된 모델이 없습니다."},
	{"", "Requested model: ", "요청 모델: "},
	{"", "Configured context budget: %d tokens", "설정 컨텍스트 예산: %d 토큰"},
	{"", "Effective context budget: %d tokens", "적용 컨텍스트 예산: %d 토큰"},
	{"", "Serving context limit: %d tokens", "서빙 컨텍스트 한도: %d 토큰"},
	{"", "Serving context limit: not reported", "서빙 컨텍스트 한도: 서버 미제공"},
	{"", "Input limit: %d tokens", "입력 한도: %d 토큰"},
	{"", "Output limit: %d tokens", "출력 한도: %d 토큰"},
	{"", "Thinking OFF: unsupported (ON only)", "추론 끄기: 서버 미지원 (켜짐 전용)"},
	{"", "Identity unavailable. Reconnect to query.", "식별 정보가 없습니다. 다시 연결하여 조회하세요."},
	{"", "Server original declaration", "서버 원본 선언"},
	{"", "Multiple models / dynamic routing", "복수 모델 / 동적 라우팅"},
	{"", "Family / architecture information only", "모델군·아키텍처 정보만 확인"},
	{"", "Original model unidentified", "원본 식별 불가"},
	{"", "Identity: ", "식별 상태: "},
	{"", "Server original ID matches catalog", "서버 원본 ID와 카탈로그 일치"},
	{"", "Serving name matches catalog", "서빙 이름과 카탈로그 일치"},
	{"", "User TOML mapping", "사용자 TOML 매핑"},
	{"", "No specialization", "특화 적용 없음"},
	{"", "Profile: ", "적용 프로파일: "},
	{"", "Profile basis: ", "프로파일 근거: "},
	{"", "Metadata unavailable (support, authorization, response or timeout).", "메타데이터를 조회할 수 없습니다 (미지원·권한·응답·시간 제한)."},
	{"", "Incomplete metadata; automatic specialization deferred.", "메타데이터가 불완전하여 자동 특화를 보류했습니다."},
	{"", "Unsupported / unverified", "미지원 / 미검증"},
	{"", "Verified by connection diagnostics", "연결 진단에서 검증됨"},
	{"", "Thinking ON/OFF: ", "추론 켜기·끄기: "},
	{"", "Advertised Thinking options and verified controls are separate.", "서버가 광고한 추론 옵션과 실제 제어 검증은 별개입니다."},
	{"", "Server declarations do not guarantee weight identity.", "서버 선언은 실제 가중치의 동일성을 보증하지 않습니다."},
	{"", "Source: ", "출처: "},
	{"", "Thinking effort: ", "추론 강도: "},
}
