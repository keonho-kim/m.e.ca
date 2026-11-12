// Purpose: Own browser failure identities and safe user-facing error presentation.
// Pattern: Central error catalog.
// Usage: API and UI actions report through this module.
// Related: webui/src/shared/api/client.ts, webui/src/pages/workspace/model/store.ts.
const errors = {
  network: '서버에 연결할 수 없습니다. 연결 상태를 확인해 주세요.',
  unauthorized: '서버 토큰으로 다시 인증해 주세요.',
  forbidden: '서버가 이 접속 주소를 허용하지 않습니다. 허용 Host·Origin 설정을 확인해 주세요.',
  browser_cookie:
    '토큰은 확인됐지만 브라우저 인증 쿠키가 저장되지 않았습니다. 쿠키 설정과 접속 주소를 확인해 주세요.',
  request_pending: '이전 제출 결과를 먼저 확인해 주세요.',
  references_pending:
    '파일 변경 결과를 확인하지 못했습니다. 파일과 첨부 경로를 다시 확인한 뒤 전송해 주세요.',
  invalid_response: '서버 응답을 해석할 수 없습니다.',
  no_session: '새 대화를 만들거나 기존 대화를 선택해 주세요.',
  no_model: '설정에서 모델을 선택해 주세요.',
  invalid_model_connection: '모델 Provider와 올바른 HTTP·HTTPS 서버 주소를 입력해 주세요.',
  input_empty: '작업 내용을 입력하거나 이미지를 첨부해 주세요.',
  input_limit: '한 요청의 본문 또는 전송 크기 한도를 초과했습니다. 내용을 나누어 보내 주세요.',
  read_only: '이 파일은 읽기 전용입니다.',
  editor_limit: '편집기 크기 한도를 초과했습니다. 붙여넣을 내용을 줄여 주세요.',
  conflict: '다른 작업이 내용을 변경했습니다. 현재 내용을 다시 확인해 주세요.',
  image_limit: '이미지 개수 또는 전체 크기 한도를 초과했습니다.',
  image_type: '현재는 이미지 파일만 첨부할 수 있습니다.',
  clipboard_empty: '클립보드에 이미지가 없습니다.',
  clipboard_unavailable:
    '클립보드 이미지를 읽을 수 없습니다. 브라우저 권한을 확인하거나 +에서 이미지를 선택해 주세요.',
  folder_picker: 'OS 폴더 선택창을 열지 못했습니다. 작업 공간 탐색기에서 폴더를 선택해 주세요.',
  folder_picker_timeout:
    'OS 폴더 선택 시간이 초과되었습니다. 작업 공간 탐색기에서 폴더를 선택해 주세요.',
  folder_picker_busy:
    'OS 폴더 선택창이 이미 열려 있습니다. 기존 창에서 선택하거나 작업 공간 탐색기를 사용해 주세요.',
  queue_limit: '입력 대기열의 메모리 한도를 초과했습니다.',
  draft_changed: '작성 중인 내용을 저장하거나 버린 뒤 이동해 주세요.',
  cancelled: '요청을 취소했습니다.',
} as const;
export type ErrorCode = keyof typeof errors;
export class ApiError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly rejected = false
  ) {
    super(message);
    this.name = 'MecaError';
  }
}
export function failure(code: ErrorCode) {
  return new ApiError(code, errors[code]);
}
export function describeError(error: unknown) {
  if (error instanceof Error) return error.message;
  return errors.network;
}

// Browser-owned request cancellation is expected. A server "canceled" result
// can describe partial filesystem effects and must still reach the user.
export function isCancelled(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError';
}
