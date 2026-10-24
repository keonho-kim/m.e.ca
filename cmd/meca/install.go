// Purpose: Wire noninteractive installation commands to their application owners.
// Pattern: Composition Root.
// Usage: start dispatches install and uninstall before bootstrap or terminal setup.
// Related: cmd/meca/main.go, internal/commands/install/install.go, internal/config/credentials/keyring.go.
package main

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/commands/install"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"os"
)

func maintenance(ctx context.Context, action startup.Action, home string) error {
	if action == startup.Uninstall {
		if err := install.Uninstall(ctx, home, credentials.Clear); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "MECA 삭제 완료: 설치 실행 파일, PATH 등록, 저장 데이터와 인증정보를 정리했습니다.")
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	result, err := install.Install(ctx, install.Options{Home: home, Executable: executable, Shell: os.Getenv("SHELL"), Path: os.Getenv("PATH")})
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "MECA 설치 완료:", result.Binary)
	if result.RestartShell {
		fmt.Fprintln(os.Stdout, "새 터미널에서 meca를 실행하세요. 현재 터미널에 반영: exec \"$SHELL\" -l")
	} else {
		fmt.Fprintln(os.Stdout, "meca 명령으로 시작하세요.")
	}
	return nil
}
