// Purpose: Present MECA's geometric lettering as an accessible connection-screen heading.
// Pattern: Shared brand presentation with font-independent vector artwork.
// Usage: Login and connection loading render this instead of repeating an icon and title.
// Related: webui/src/pages/connect/ui/connect-page.tsx, webui/src/pages/connect/ui/connection-loading.tsx.
import { Heading } from '@astryxdesign/core/Heading';

export function Wordmark() {
  return (
    <Heading level={1} aria-label="MECA" className="min-w-0">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 496 96"
        fill="currentColor"
        className="block h-auto w-60 max-w-full"
        aria-hidden="true"
        focusable="false"
      >
        <path d="M0 96V0h20l32 34L84 0h20v96H84V32L52 66 20 32v64z" />
        <path d="M132 0h94v20h-72v18h66v20h-66v18h72v20h-94z" />
        <path d="M274 0h74v20h-64l-8 8v40l8 8h64v20h-74l-20-20V20z" />
        <path fillRule="evenodd" d="M420 0h24l52 96h-24l-10-20h-60l-10 20h-24zm-8 56h40l-20-36z" />
      </svg>
    </Heading>
  );
}
