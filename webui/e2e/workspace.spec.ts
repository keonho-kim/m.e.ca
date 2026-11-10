// Purpose: Verify native multi-workspace execution, editing and settings through the built embedded UI.
// Pattern: Browser-to-Go integration contracts with a free deterministic model fixture.
// Usage: Start TestUIBrowserFixture then run bun run test:e2e with the same descriptor path.
// Related: cmd/meca/ui_browser_test.go, cmd/meca/ui_provider_test.go.
import {readFile,writeFile,mkdir} from 'node:fs/promises';
import {join} from 'node:path';
import {test,expect,type Page} from '@playwright/test';
import type {CommandJson} from '@/shared/api';
import type {ReplyJson} from '@/shared/api';

let fixture:{url:string;tokenFile:string;projectA:string;projectB:string;home:string;provider:string;archiveSession:string};
test.describe.configure({mode:'serial'});
test.beforeAll(async()=>{
 const path=process.env.MECA_UI_DESCRIPTOR;
 test.skip(!path,'start the isolated Go UI fixture first');
 fixture=JSON.parse(await readFile(path!,'utf8'));
});
test.beforeEach(async({page})=>{
 await page.goto(`${fixture.url}/ui/`);
 await page.getByRole('textbox',{name:'서버 토큰',exact:true}).fill((await readFile(fixture.tokenFile,'utf8')).trim());
 await page.getByRole('button',{name:'연결',exact:true}).click();
 await page.locator('.workspace-name').filter({hasText:'alpha'}).waitFor();
});
async function rpc(page:Page,command:CommandJson,workspace=''):Promise<ReplyJson>{
 return page.evaluate(async({command,workspace})=>{
  const response=await fetch('/v1/command',{method:'POST',headers:{'Content-Type':'application/json','X-Meca-UI':'1'},body:JSON.stringify({...command,workspaceId:workspace})});
  const reply=await response.json();if(reply.error)throw Error(`${reply.error.code}: ${reply.error.message}`);return reply;
 },{command,workspace});
}
const input=(page:Page)=>page.getByLabel('작업 입력',{exact:true});
async function create(page:Page,workspace='alpha'){
 await page.locator('.workspace-name').filter({hasText:workspace}).click();
 const group=page.locator('.workspace-group').filter({has:page.locator('.workspace-name',{hasText:workspace})});
 await group.getByRole('button',{name:'새 대화',exact:true}).click();
 await expect(page).toHaveURL(/\/c\/session_/);
 await expect(input(page)).toBeEnabled();
 return page.url();
}
async function send(page:Page,text:string){await input(page).fill(text);await page.getByRole('button',{name:/^(전송|보내기)$/,exact:true}).click();}

test('an empty conversation has one composer with an upward image and skill accordion',async({page})=>{
 await create(page);
 await expect(page.getByRole('heading',{name:'어떤 작업을 시작할까요?',exact:true})).toBeVisible();
 await expect(input(page)).toBeEnabled();
 const composer=page.locator('.composer'),plus=composer.getByRole('button',{name:'첨부 및 스킬',exact:true});
 await expect(composer.getByRole('button',{name:/^모델 /})).toHaveCount(0);
 await expect(composer.getByRole('button',{name:/받아쓰기/})).toHaveCount(0);
 await expect(page.getByRole('button',{name:'스킬 목록 보기',exact:true})).toBeHidden();
 await plus.click();await expect(plus).toHaveAttribute('aria-expanded','true');
 await expect(page.getByRole('button',{name:'스킬 목록 보기',exact:true})).toBeVisible();
 await expect(page.locator('input[type=file]')).toHaveAttribute('accept','image/*');
 await expect.poll(()=>plus.evaluate(node=>getComputedStyle(node).transform)).toMatch(/matrix\(0, 1, -1, 0/);
 const panel=page.locator(`[id=${JSON.stringify(await plus.getAttribute('aria-controls'))}]`);
 expect((await panel.boundingBox())!.y).toBeLessThan((await plus.boundingBox())!.y);
 await page.keyboard.press('Escape');await expect(plus).toHaveAttribute('aria-expanded','false');
 await expect(page.getByRole('button',{name:'스킬 목록 보기',exact:true})).toBeHidden();
 await send(page,'Astryx composition contract');
 await expect(page.getByRole('heading',{name:'어떤 작업을 시작할까요?',exact:true})).toBeHidden();
 await expect(page.locator('.history-rows')).toContainText('Astryx composition contract');
});

test('light is the default, appearance persists and reduced motion keeps controls usable',async({page})=>{
 await page.emulateMedia({colorScheme:'dark'});
 await expect(page.locator('html')).toHaveAttribute('data-theme','light');
 const shortcut=()=>page.getByRole('button',{name:'화면 테마',exact:true});
 await shortcut().click();await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 await page.reload();await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 await shortcut().click();await expect(page.locator('html')).toHaveAttribute('data-theme','light');
 await page.getByRole('button',{name:'설정',exact:true}).click();
 const modal=page.getByRole('dialog');
 await modal.getByRole('button',{name:'일반',exact:true}).click();
 const themes=modal.getByRole('radiogroup',{name:'화면 테마',exact:true});
 await themes.getByRole('radio',{name:'라이트',exact:true}).focus();
 await page.keyboard.press('ArrowRight');
 await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 await themes.getByRole('radio',{name:'시스템',exact:true}).click();
 await page.emulateMedia({colorScheme:'light',reducedMotion:'reduce'});
 await expect(page.locator('html')).toHaveAttribute('data-theme','light');
 await expect(modal.getByRole('radio',{name:'시스템',exact:true})).toBeChecked();
 expect(await modal.evaluate((node)=>getComputedStyle(node).animationName)).toBe('none');
 await page.keyboard.press('Escape');await expect(modal).toBeHidden();
 await create(page);
 const sendButton=page.getByRole('button',{name:/^(전송|보내기)$/,exact:true});
 await sendButton.hover();await page.mouse.down();
 expect(await sendButton.evaluate(node=>node.matches(':active'))).toBe(true);
 expect(await sendButton.evaluate(node=>{const value=getComputedStyle(node).transform;return value==='none'||new DOMMatrix(value).isIdentity;})).toBe(true);
 await page.mouse.move(0,0);await page.mouse.up();
});

test('multiple projects continue independently and one completed predecessor runs a follow-up once',async({page})=>{
 const first=await create(page);await send(page,'HOLD_FIXTURE alpha first');
 await expect(page.getByRole('button',{name:'작업 중지',exact:true})).toBeVisible();
 await input(page).fill('Captured follow-up once');await page.getByRole('button',{name:'후속 예약',exact:true}).click();
 await expect(page.getByRole('region',{name:'예약 요청'})).toBeVisible();
 await create(page);await send(page,'HOLD_FIXTURE alpha second');
 await expect(page.getByRole('button',{name:'작업 중지',exact:true})).toBeVisible();
 await create(page,'beta');await send(page,'HOLD_FIXTURE beta first');
 await expect(page.getByRole('button',{name:'작업 중지',exact:true})).toBeVisible();
 expect((await rpc(page,{listWorkspaces:{}})).workspaces?.items?.reduce((n,w)=>n+(w.activeTasks??0),0)).toBe(3);
 await page.getByRole('button',{name:'스트림 일시정지',exact:true}).click();
 await page.getByRole('button',{name:'스트림 재연결',exact:true}).click();
 await expect(page.locator('.live-output')).toContainText('Fixture');
 await page.getByRole('button',{name:'작업 중지',exact:true}).click();
 await expect(page.getByRole('button',{name:'작업 중지',exact:true})).toBeHidden();
 await page.request.post(`${fixture.provider}/test/release`);
 await page.locator('.session-link').filter({hasText:'HOLD_FIXTURE alpha first'}).click();
 await expect(page.locator('.history-rows')).toContainText('Captured follow-up once');
 const parts=new URL(first).pathname.split('/');
 await expect.poll(async()=> (await rpc(page,{listTasks:{sessionId:parts[5]}},parts[3])).tasks?.items?.length).toBe(2);
 await expect(page.getByRole('region',{name:'예약 요청'})).toBeHidden();
});

test('files use complete revision-bound saves and skills insert canonical mentions',async({page})=>{
 await writeFile(join(fixture.projectA,'sample.go'),'package sample\n\nfunc Value() int { return 1 }\n');
 await create(page);await send(page,'!dir');
 await page.getByRole('checkbox',{name:'선택 참조: sample.go',exact:true}).check();
 await page.getByRole('button',{name:'sample.go',exact:true}).click();
 await page.getByRole('button',{name:'파일 편집',exact:true}).click();
 await page.getByRole('textbox',{name:/sample\.go$/}).fill('package sample\n\nfunc Value() int { return 42 }\n');
 await page.getByRole('button',{name:'저장',exact:true}).click();
 await expect(page.getByRole('button',{name:'저장',exact:true})).toBeDisabled();
 await expect.poll(()=>readFile(join(fixture.projectA,'sample.go'),'utf8')).toContain('return 42');
 await page.getByRole('textbox',{name:/sample\.go$/}).fill('package sample\n\nfunc Value() int { return 43 }\n');
 await writeFile(join(fixture.projectA,'sample.go'),'package sample\n\nfunc Value() int { return 99 }\n');
 await page.getByRole('button',{name:'저장',exact:true}).click();
 await expect(page.getByText('file changed',{exact:false})).toBeVisible();
 await expect(page.getByRole('textbox',{name:/sample\.go$/})).toContainText('return 43');
  await page.locator('.document-toolbar').getByRole('button',{name:'취소',exact:true}).click();await page.getByRole('button',{name:'변경 버리기',exact:true}).click();
 await page.locator('.workspace-header').getByRole('tab',{name:'스킬',exact:true}).click();
 await page.locator('.skill-entry').filter({hasText:'ui-design'}).click();
 await expect(page.locator('.skill-document')).toContainText('ui-design');
 await page.getByRole('button',{name:'입력에 스킬 넣기',exact:true}).click();
 await expect(input(page)).toHaveText(/#meca:ui-design/);
 await send(page,'!model');
 const modal=page.getByRole('dialog');
 await expect(modal.getByRole('combobox',{name:'Provider',exact:true})).toContainText('vLLM');
 await modal.getByRole('button',{name:'일반',exact:true}).click();
 await modal.getByRole('radio',{name:'다크',exact:true}).click();
 await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 await modal.getByRole('combobox',{name:'언어',exact:true}).click();await page.getByRole('option',{name:'English',exact:true}).click();
 await expect(modal.getByRole('heading',{name:'Settings',exact:true})).toBeVisible();
 await modal.getByRole('combobox',{name:'Language',exact:true}).click();await page.getByRole('option',{name:'한국어',exact:true}).click();
 await modal.getByRole('button',{name:'닫기',exact:true}).click();
 await page.setViewportSize({width:390,height:844});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('folder registration, authenticated images and task-bound approval are operational',async({page})=>{
 const added=join(fixture.home,'gamma');await mkdir(added,{recursive:true});
 await page.getByRole('button',{name:'워크스페이스 추가',exact:true}).click();
 let modal=page.getByRole('dialog',{name:'워크스페이스 추가',exact:true});
 await modal.getByRole('textbox',{name:'경로',exact:true}).fill(added);
 await modal.getByRole('button',{name:'열기',exact:true}).click();
 await expect(modal.locator('.folder-current')).toContainText('gamma');
 await modal.getByRole('button',{name:'폴더 선택',exact:true}).click();
 await expect(page.locator('.workspace-name').filter({hasText:'gamma'})).toBeVisible();
 await create(page);
 await page.getByRole('button',{name:'첨부 및 스킬',exact:true}).click();await page.locator('input[type=file]').setInputFiles(join(fixture.home,'fixture.png'));
 await send(page,'Inspect the attached fixture');
 const image=page.getByRole('group',{name:'이미지 첨부',exact:true}).first();
 await expect(image).toBeVisible();
 await image.getByRole('button').click();
 modal=page.getByRole('dialog');
 await expect.poll(()=>modal.locator('img').evaluate((image:HTMLImageElement)=>image.naturalWidth)).toBe(16);
 await modal.getByRole('button',{name:'닫기',exact:true}).click();
 await create(page);await send(page,'APPROVAL_FIXTURE');
 modal=page.getByRole('dialog',{name:'승인 요청',exact:true});
 await expect(modal).toBeVisible();
 await modal.getByRole('button',{name:'승인',exact:true}).click();
 await expect(modal).toBeHidden();
 await expect(page.locator('.history-rows')).toContainText('Completed: APPROVAL_FIXTURE');
 expect(await readFile(join(fixture.home,'approval-written.txt'),'utf8')).toBe('approved fixture\n');
});


test('long history keeps a bounded page window and can return to the latest saved response',async({page})=>{
 const workspaces=(await rpc(page,{listWorkspaces:{}})).workspaces!;
 await page.goto(`${fixture.url}/ui/w/${workspaces.defaultId}/c/${fixture.archiveSession}`);
 await expect(page.locator('.history-rows')).toContainText('Archive message 299');
 for(let index=0;index<4;index++){
  const response=page.waitForResponse(response=>response.url().endsWith('/v1/command')&&!!response.request().postDataJSON()?.history);
  await page.getByRole('tabpanel',{name:'대화',exact:true}).getByRole('button',{name:'이전 이력 더 보기',exact:true}).click();await response;
 }
 await page.getByRole('button',{name:'최근 대화로 돌아가기',exact:true}).click();
 await expect(page.locator('.history-rows')).toContainText('Archive message 299');
 expect(await page.locator('.history-row').count()).toBeLessThan(50);
});


test('deleting an idle conversation removes its saved content and browser navigation cannot restore it',async({page})=>{
 await create(page);await send(page,'Delete this fixture conversation');
 await expect(page.locator('.history-rows')).toContainText('Completed: Delete this fixture conversation');
 await expect(page.getByRole('button',{name:'작업 중지',exact:true})).toBeHidden();
 await page.locator('.session-row[aria-current=page]').locator('..').getByRole('button',{name:'대화 삭제',exact:true}).click();
 await page.getByRole('dialog',{name:'대화 삭제',exact:true}).getByRole('button',{name:'대화 삭제',exact:true}).click();
 await expect(page).not.toHaveURL(/\/c\/session_/);
 await page.goBack();
 await expect(page.locator('.history-rows').filter({hasText:'Delete this fixture conversation'})).toHaveCount(0);
});

test('composer accepts image drops and native and explicit clipboard shortcuts without duplicate attachment',async({page})=>{
 await create(page);
 const bytes=(await readFile(join(fixture.home,'fixture.png'))).toString('base64');
 const transfer=await page.evaluateHandle(bytes=>{
  const data=new DataTransfer();
  data.items.add(new File([Uint8Array.from(atob(bytes),c=>c.charCodeAt(0))],'drop.png',{type:'image/png'}));
  return data;
 },bytes);
 await page.locator('.composer').dispatchEvent('drop',{dataTransfer:transfer});
 await expect(page.locator('.composer').getByText('drop.png',{exact:true})).toHaveCount(1);
 await input(page).dispatchEvent('drop',{dataTransfer:transfer});
 await expect(page.locator('.composer').getByText('drop.png',{exact:true})).toHaveCount(2);
 await transfer.dispose();
 await input(page).focus();
 await input(page).evaluate((editor,bytes)=>{
  const data=new DataTransfer();
  data.items.add(new File([Uint8Array.from(atob(bytes),c=>c.charCodeAt(0))],'paste.png',{type:'image/png'}));
  editor.dispatchEvent(new ClipboardEvent('paste',{clipboardData:data,bubbles:true,cancelable:true}));
 },bytes);
 await expect(page.locator('.composer').getByText('paste.png',{exact:true})).toHaveCount(1);
 await page.evaluate(bytes=>{
  const image=new Blob([Uint8Array.from(atob(bytes),c=>c.charCodeAt(0))],{type:'image/png'});
  Object.defineProperty(navigator,'clipboard',{configurable:true,value:{read:async()=>[{types:['image/png'],getType:async()=>image}]}});
  Object.defineProperty(navigator,'platform',{configurable:true,value:'MacIntel'});
 },bytes);
 await input(page).focus();await page.keyboard.press('Alt+KeyV');
 await expect(page.locator('.composer').getByText('clipboard-1.png',{exact:true})).toHaveCount(1);
 await page.keyboard.press('Control+KeyV');
 await expect(page.locator('.composer').getByText('clipboard-1.png',{exact:true})).toHaveCount(2);
 await expect(page.getByRole('button',{name:'보내기',exact:true})).toBeEnabled();
});

test('thinking streams as subdued Markdown and remains readable in saved history',async({page})=>{
 await create(page);await send(page,'THINKING_FIXTURE');
 const thinking=page.locator('.thinking');
 await expect(thinking.getByRole('heading',{name:'Plan',exact:true})).toBeVisible();
 await expect(thinking.locator('strong')).toHaveText('Inspect');
 await expect(thinking).toContainText('the workspace');
 await expect(page.locator('.history-rows')).not.toContainText('Completed: THINKING_FIXTURE');
 for(const theme of ['light','dark']){
  await expect(page.locator('html')).toHaveAttribute('data-theme',theme);
  expect(await thinking.locator('[data-tone=thinking]').evaluate(el=>{
   const style=getComputedStyle(el),reference=document.createElement('span');
   reference.style.color='var(--color-text-secondary)';el.append(reference);
   const expected=getComputedStyle(reference).color;reference.remove();
   return style.color===expected;
  })).toBe(true);
  if(theme==='light')await page.getByRole('button',{name:'화면 테마',exact:true}).click();
 }
 await page.request.post(`${fixture.provider}/test/release-thinking`);
 await expect(page.locator('.history-rows')).toContainText('Completed: THINKING_FIXTURE');
 await expect(page.locator('.history-rows .thinking')).toContainText('and verify the result.');
 await page.reload();await expect(page.locator('.history-rows .thinking')).toContainText('and verify the result.');
});

test('mobile settings give the model form full width and keep discovery diagnosis and save usable',async({page})=>{
 await create(page);await page.setViewportSize({width:390,height:844});
 await page.getByRole('button',{name:'열기',exact:true}).click();
 await page.getByRole('button',{name:'설정',exact:true}).click();
 const dialog=page.getByRole('dialog',{name:'설정',exact:true});
 await expect(dialog.getByRole('combobox',{name:'설정 항목',exact:true})).toBeVisible();
 await expect.poll(async()=>(await dialog.getByRole('textbox',{name:'Base URL',exact:true}).boundingBox())?.width??0).toBeGreaterThan(280);
 for(const [label,key] of [['모델 조회','discoverModels'],['연결 진단','diagnoseModel'],['저장','setModel']] as const){
  const reply=page.waitForResponse(response=>response.url().endsWith('/v1/command')&&response.request().postDataJSON()?.[key]!==undefined);
  await dialog.getByRole('button',{name:label,exact:true}).click();
  expect((await(await reply).json()).error).toBeUndefined();
 }
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});

test('long live Markdown follows the newest output after durable history has settled',async({page},info)=>{
 await page.emulateMedia({reducedMotion:'reduce'});await create(page);await send(page,'LONG_STREAM_FIXTURE');
 await expect(page.locator('.history-rows')).toContainText('LONG_STREAM_FIXTURE');
 await expect(page.locator('.live-output')).toContainText('Fixture');
 const scroller=page.locator('.workspace-body');
 await scroller.evaluate(el=>{el.scrollTop=el.scrollHeight});
 const gap=()=>scroller.evaluate(el=>el.scrollHeight-el.clientHeight-el.scrollTop);
 await expect.poll(gap).toBeLessThan(30);
 await page.request.post(`${fixture.provider}/test/expand-stream`);
 await expect.poll(()=>scroller.evaluate(el=>el.scrollHeight)).toBeGreaterThan(3000);
 await expect.poll(gap).toBeLessThan(30);
 await page.screenshot({path:info.outputPath('streaming-long.png'),fullPage:true,animations:'disabled'});
 await page.getByRole('button',{name:'작업 중지',exact:true}).click();
 await expect(page.getByRole('button',{name:'작업 중지',exact:true})).toBeHidden();
});

test('explicit disconnect releases private editor state after the dirty-file confirmation',async({page})=>{
 await create(page);await send(page,'!dir');
 await page.getByRole('button',{name:'sample.go',exact:true}).click();
 await page.getByRole('button',{name:'파일 편집',exact:true}).click();
 await page.getByRole('textbox',{name:/sample\.go$/}).fill('private unsaved fixture');
 await page.getByRole('button',{name:'설정',exact:true}).click();
 const settings=page.getByRole('dialog',{name:'설정',exact:true});
 await settings.getByRole('button',{name:'일반',exact:true}).click();
 await settings.getByRole('button',{name:'연결 종료',exact:true}).click();
 await expect(page.getByRole('alertdialog')).toBeVisible();
 await page.getByRole('alertdialog').getByRole('button',{name:'계속 편집',exact:true}).click();
 await expect(page.getByRole('textbox',{name:/sample\.go$/})).toHaveCount(1);
 await settings.getByRole('button',{name:'연결 종료',exact:true}).click();
 await page.getByRole('alertdialog').getByRole('button',{name:'연결 종료',exact:true}).click();
 await expect(page.getByRole('textbox',{name:'서버 토큰',exact:true})).toBeVisible();
 await expect(page.locator('.text-editor')).toHaveCount(0);
 await expect(page.getByText('private unsaved fixture',{exact:false})).toHaveCount(0);
});
