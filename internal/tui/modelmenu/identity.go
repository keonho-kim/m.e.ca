// Purpose: Project selected model identity and verified controls into menu labels.
// Pattern: Presentation projection.
// Usage: Terminal menus pass an immutable selection snapshot and redactor.
// Related: internal/provider/identity/report.go, internal/tui/session/render_menu.go.
package modelmenu

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/identity"
)

func IdentifiedName(selected connection.Selection, redact func(string) string) string {
	report := selected.Identity
	if report == nil || report.Unavailable || report.Incomplete || report.State() != "reported" {
		return ""
	}
	name := identity.Repository(report.Originals()[0])
	if name == selected.Model {
		return ""
	}
	return redact(name)
}

func IdentityLines(selected connection.Selection, contextLength int, redact func(string) string) []string {
	if selected.Model == "" {
		return []string{"연결된 모델이 없습니다."}
	}
	lines := []string{"요청 모델: " + redact(selected.Model), fmt.Sprintf("설정 컨텍스트 예산: %d 토큰", contextLength)}
	reported := 0
	if selected.Details != nil {
		reported = selected.Details.ContextWindow
	}
	lines = append(lines, fmt.Sprintf("적용 컨텍스트 예산: %d 토큰", connection.EffectiveContextLength(contextLength, reported)))
	if info := selected.Details; info != nil {
		if info.ContextWindow > 0 {
			lines = append(lines, fmt.Sprintf("서빙 컨텍스트 한도: %d 토큰", info.ContextWindow))
		} else {
			lines = append(lines, "서빙 컨텍스트 한도: 서버 미제공")
		}
		if info.MaxInput > 0 {
			lines = append(lines, fmt.Sprintf("입력 한도: %d 토큰", info.MaxInput))
		}
		if info.MaxOutput > 0 {
			lines = append(lines, fmt.Sprintf("출력 한도: %d 토큰", info.MaxOutput))
		}
		if info.Notice != "" {
			lines = append(lines, redact(info.Notice))
		}
		if info.Reasoning != nil && info.Reasoning.Enabled() && !info.Reasoning.Off {
			lines = append(lines, "Thinking OFF: 서버 미지원 (ON 전용)")
		}
	}
	report := selected.Identity
	if report == nil {
		return append(lines, "식별 정보가 없습니다. 다시 연결하여 조회하세요.")
	}
	states := map[string]string{"reported": "서버 원본 선언", "ambiguous": "복수 모델 / 동적 라우팅", "family": "모델군·아키텍처 정보만 확인", "unknown": "원본 식별 불가"}
	lines = append(lines, "식별 상태: "+states[report.State()])
	sources := map[string]string{"server": "서버 원본 ID와 카탈로그 일치", "catalogue": "서빙 이름과 카탈로그 일치", "manual": "사용자 TOML 매핑", "generic": "특화 적용 없음"}
	lines = append(lines, "적용 프로파일: "+selected.Profile, "프로파일 근거: "+sources[report.ProfileSource])
	if report.Unavailable {
		lines = append(lines, "메타데이터를 조회할 수 없습니다 (미지원·권한·응답·시간 제한).")
		if report.Failure != "" {
			lines = append(lines, redact(report.Failure))
		}
	}
	if report.Incomplete {
		lines = append(lines, "메타데이터가 불완전하여 자동 특화를 보류했습니다.")
	}
	control := "미지원 / 미검증"
	if selected.ThinkingSupported {
		control = "연결 진단에서 검증됨"
	}
	lines = append(lines, "Thinking ON/OFF: "+control, "서버가 광고한 Thinking 옵션과 실제 제어 검증은 별개입니다.", "서버 선언은 실제 가중치의 동일성을 보증하지 않습니다.", "")
	for _, fact := range report.Facts {
		lines = append(lines, "  "+fact.Field+": "+redact(fact.Value), "    출처: "+fact.Source)
	}
	return lines
}
