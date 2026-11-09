# Korean-language desktop application design

Use this when the product serves Korean-language desktop users. These are contextual design considerations, not a claim that all Korean products share one style. Enterprise operations, public information, consumer services and promotional pages need different density and expression. Desktop workflow and language evidence matter more than copying a mobile brand's visual treatment.

## Compose familiar working regions when useful

For operational products, consider a stable global header, local navigation, clear page title/context, a query area, results and a detail/edit destination. GNB commonly names global navigation and LNB local navigation; explain their roles rather than making the user learn the abbreviations.

A repeated-query workflow may benefit from related conditions grouped above a result table, with 조회/검색 and 초기화 clearly separated. Keep ordinary conditions visible and disclose genuinely less common ones. An explicit Apply action suits several dependent or expensive filters; immediate filtering suits other tasks. Preserve the distinction between edited conditions and the results currently displayed.

Useful desktop roles include 목록 for return, 상세 for a record view, 등록 for creating an object and 저장 for committing an edit, where those meanings match the product. Do not use 확인 as a vague label for every action. Match the action and resulting feedback consistently.

## Density and navigation

- Comparative work often benefits from aligned tables, stable column meanings, visible sort state and contextual row actions.
- Dense does not mean tiny. Reduce redundant chrome and spacing before reducing functional text or making controls difficult to target.
- Keep important selection, result count and applied scope explicit. Distinguish current-page selection from all matching records.
- A list-detail split can preserve context for repeated inspection. A separate route is better for long independent work; choose from the task.
- Use breadcrumbs for actual hierarchy and local sections for substantial details. Do not add a breadcrumb and repeated title at every nesting level without useful information.
- A wide desktop can use its space for meaningful columns or simultaneous context. A promotional reading page can instead keep a capped container; these are different needs.

These patterns are useful in some Korean business applications, not mandatory proof of Korean identity. Avoid replicating cluttered portals, excessive popup windows or several competing toolbars as a national convention.

## Hangul, Latin and numbers

Use fonts with real Hangul coverage and inspect mixed Korean, Latin, numerals, punctuation and units together. Reuse available project fonts. A system's use of Pretendard, Noto Sans or another family does not make that font a universal requirement or authorize a new remote dependency.

Check the actual visual size, weight and line height; the same nominal size can read differently across fonts. Do not copy Latin-only cap-height or extreme negative-tracking treatments onto Korean headings. Reserve expressive display treatment for an appropriate subject, while navigation, data and labels remain readable.

Test real long words and mixed identifiers. `word-break: keep-all` may help Korean phrase wrapping in suitable regions, but can cause overflow with long unbroken content. Pair the intended wrapping policy with a safe overflow strategy and inspect actual headings, tables and identifiers. Do not apply one breaking policy to every code block, URL and prose paragraph.

Use truncation only where the layout needs it and provide an accessible route to full content. Keep units close to the values they explain. Align comparable amounts and use tabular numerals where the font supports them.

## Input and local formatting

Preserve IME composition in search, autocomplete and shortcuts. Enter used to finish composition must not also trigger a candidate or submit a mutation. Test keyboard operation, pasted content and mixed-script input, not just Latin typing.

Format dates, amounts, units and phone/address displays using the product's locale and data contract. Distinguish 원, 천 원 and 만 원 explicitly when used. A timestamp's timezone and an account's operating timezone may differ; do not assume every Korean-language user or transaction is in Korea.

Only request personal or business information the actual workflow requires. Do not add identity numbers, consent boxes, a company registration number or certification claims to imitate a local service. Footer contact and business facts must come from supplied truth.

## Give Korean content an intentional hierarchy

A restrained tool area and an expressive article or authored artifact can coexist. Choose display treatment from the content role rather than importing a Latin specimen or making every page title large. Verify the actual Korean fallback and mixed-script rhythm. Serif can serve editorial reading; it is not required for every Korean content product.

Use accent where action, current context, content category or actual state needs recognition. Neutral backgrounds do not require faint labels, and colorful content does not require equally colorful chrome. Preserve readable secondary facts, units and complete object identity; professional appearance comes from consistent hierarchy and operation as much as color.

Separate global navigation from project modes, object outlines and result views in the user's own language. A familiar data-oriented frame may help repeated work, while a media/reading task can lead differently. Do not reproduce the bundled studies' menu order, fonts or dimensions as a national convention.

## Themes and quality

Any of the selected material themes can work with Korean content. Preserve readable Hangul, semantic status, visible focus and actual task regions beneath glass or futuristic expression. Minimalism should not hide essential conditions and operations; information density alone is not slop.

Inspect a laptop-sized desktop, a larger display, a resized window and enlarged text. Keep dropdowns, row actions, sticky headers and action bars reachable. A mobile screenshot cannot establish that the desktop comparison or input workflow works.

## Source context

KRDS documents grids, margins, typography and task/service patterns for digital government. These inform practical design questions, not a requirement to adopt government branding or all public-sector rules in a private SaaS:

- Layout: https://www.krds.go.kr/html/site/style/style_05.html
- Typography: https://www.krds.go.kr/html/site/style/style_03.html
- Pattern context: https://www.krds.go.kr/html/site/outline/outline_02.html

For evidence that desktop tools need their own task and component reasoning rather than transplanted mobile surfaces, see Toss's tools-design account: https://toss.tech/article/1st-product-designer-tools
