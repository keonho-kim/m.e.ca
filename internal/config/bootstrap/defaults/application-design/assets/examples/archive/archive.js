// Purpose: Connect a content-first library, asset viewer and learning pages with scoped local state.
// Pattern: Native page controller with bounded image ingestion.
// Usage: Archive subpages filter existing cards, preserve asset identity and store one local upload.
// Related: internal/config/bootstrap/defaults/application-design/assets/examples/archive/archive.css, internal/config/bootstrap/defaults/application-design/assets/examples/shared/media.js, internal/config/bootstrap/defaults/application-design/assets/examples/shared/local-state.js.
(() => {
  const ui=ExampleUI,options={signal:ui.signal},page=document.body.dataset.page;
  const initial={version:1,saved:['A-02'],custom:null,nextId:100,comments:{},completed:[1]};
  const assets=[
    {id:'A-01',title:'창가 스튜디오 콘셉트',kind:'image',src:'../shared/studio-daylight.jpg',preview:'../shared/daylight-preview.jpg',file:'window-studio.jpg',size:'553 KB',date:'2026.11.07',owner:'김민서'},
    {id:'A-02',title:'미팅룸 머티리얼',kind:'image',src:'../shared/studio-warm.jpg',preview:'../shared/warm-preview.jpg',file:'meeting-material.jpg',size:'546 KB',date:'2026.11.06',owner:'이서연'},
    {id:'A-03',title:'공간 프로젝트 이용 가이드',kind:'guide',file:'workspace-guide',date:'2026.11.05',owner:'김민서',subtitle:'목적, 정보 관계와 피드백'},
    {id:'A-04',title:'사용성으로 시작하는 디자인',kind:'course',file:'application-class',date:'2026.11.04',owner:'정하린',subtitle:'4개의 짧은 학습'},
    {id:'A-05',title:'프로젝트 검토 메모',kind:'note',file:'review-notes.md',date:'2026.11.03',owner:'이서연',subtitle:'결정과 근거를 함께 남기기',body:'공간을 비교할 때 필요한 정보는 한 화면에 유지합니다. 주요 이미지의 비율을 먼저 확보하고, 가끔 확인하는 상세 조건은 보충 영역에 둡니다.\n\n내비게이션과 콘텐츠, 상태 피드백은 서로 다른 역할을 갖습니다. 애니메이션은 그 관계를 설명할 때만 사용합니다.'},
    {id:'A-06',title:'상호작용 체크리스트',kind:'note',file:'interaction-review.md',date:'2026.11.02',owner:'박지훈',subtitle:'키보드, 순서, 오류와 복귀',body:'메뉴는 키보드로 열고 닫을 수 있는가?\n\n화면을 바꿨을 때 선택과 초안이 섞이지 않는가?\n\n기다리는 중에도 현재 범위와 다음 행동을 알 수 있는가?\n\n오류가 나면 입력과 실제 상태가 보존되는가?'}
  ];
  const lessons=[['과업과 정보 관계','누가 무엇을 하려는지 먼저 정리합니다. 찾아야 할 정보와 비교할 정보를 구분하세요.'],['유형과 구성','함께 비교해야 할 정보는 가까이 두고, 순서대로 이해할 정보는 읽기 흐름을 만듭니다.'],['상태와 피드백','대기, 완료, 오류와 결과를 알 수 없음을 구분합니다. 움직임이 작업의 성공을 결정하지 않습니다.'],['검토와 개선','실제 내용과 과업으로 화면을 검토합니다. 근거 없는 영역이나 효과를 줄이고 관찰한 어려움의 원인을 수정하세요.']];
  const savedView=new URLSearchParams(location.search).get('saved')==='1';
  let state,filter='all',reader=null,uploadTask=null;
  const valid=value=>Array.isArray(value.saved)&&value.saved.length<=10&&Array.isArray(value.completed)&&value.completed.length<=4&&new Set(value.completed).size===value.completed.length&&value.completed.every(i=>Number.isInteger(i)&&i>=1&&i<=4)&&Number.isSafeInteger(value.nextId)&&value.nextId>=100&&value.comments&&Object.values(value.comments).every(items=>Array.isArray(items)&&items.length<=20&&items.every(text=>typeof text==='string'&&text.length<=1000))&&(!value.custom||value.custom.kind==='image'&&/^data:image\/(jpeg|png|webp);base64,/.test(value.custom.src)&&value.custom.src.length<1500000&&/^data:image\/jpeg;base64,/.test(value.custom.preview)&&value.custom.preview.length<30000);
  const commit=next=>{ExampleState.write('archive',next);state=next;};
  const all=()=>state.custom?[...assets,state.custom]:assets;
  function figure(asset){
    const frame=document.createElement('div');frame.className='media';
    if(/^A-\d+$/.test(asset.id)&&asset.src.startsWith('../shared/'))frame.style.viewTransitionName='archive-visual-'+asset.id;
    const preview=new Image();preview.className='preview';preview.src=asset.preview;preview.alt='';preview.setAttribute('aria-hidden','true');
    const full=new Image();full.className='full';full.dataset.blurUp='';full.src=asset.src;full.alt=asset.title;full.width=1448;full.height=1086;full.decoding='async';
    const fallback=document.createElement('span');fallback.className='fallback';fallback.hidden=true;fallback.setAttribute('role','status');fallback.textContent=asset.title+'의 이미지를 불러오지 못했습니다.';
    frame.append(preview,full,fallback);return frame;
  }
  function destination(asset){return asset.kind==='guide'?'archive-guide.html':asset.kind==='course'?'archive-lesson.html':'archive-asset.html?id='+encodeURIComponent(asset.id)+(savedView?'&collection=saved':'');}
  function applyFilter(){
    const q=document.querySelector('[data-library-query]')?.value.trim().toLocaleLowerCase()||'';
    const focused=document.activeElement?.closest('[data-asset-id]');
    let count=0;
    for(const card of document.querySelectorAll('[data-asset-id]')){
      const asset=all().find(a=>a.id===card.dataset.assetId);
      card.hidden=!asset.title.toLocaleLowerCase().includes(q)||(savedView&&!state.saved.includes(asset.id))||(filter==='images'&&asset.kind!=='image')||(filter==='documents'&&asset.kind==='image');
      if(!card.hidden)count++;
      const save=card.querySelector('[data-save]');if(save){save.setAttribute('aria-pressed',String(state.saved.includes(asset.id)));save.setAttribute('aria-label',asset.title+(state.saved.includes(asset.id)?' 저장 해제':' 저장'));}
    }
    document.querySelector('[data-library-count]').textContent=`${count}개의 자료`;
    document.querySelector('[data-library-empty]').hidden=count!==0;
    for(const button of document.querySelectorAll('.filter-chips [data-library-filter]'))button.setAttribute('aria-pressed',String(button.dataset.libraryFilter===filter));
    for(const link of document.querySelectorAll('[data-collection]')){if((link.dataset.collection==='saved')===savedView)link.setAttribute('aria-current','page');else link.removeAttribute('aria-current');}
    document.querySelector('h1').textContent=savedView?'저장한 자료':'자료 라이브러리';
    const uploadButton=document.querySelector('.collection-head [data-open]');if(uploadButton)uploadButton.hidden=savedView;
    document.querySelector('[data-saved-collection-home]').hidden=!savedView;
    const empty=document.querySelector('[data-library-empty]');
    const noSaved=savedView&&state.saved.length===0;
    empty.querySelector('h2').textContent=noSaved?'아직 저장한 자료가 없습니다':'조건에 맞는 자료가 없습니다';
    empty.querySelector('[data-empty-library-copy]').textContent=noSaved?'자료에서 저장 버튼을 누르면 이곳에 모아 볼 수 있습니다.':'검색어나 자료 종류를 바꿔보세요.';
    empty.querySelector('[data-empty-library-link]').hidden=!savedView;
    const reset=empty.querySelector('[data-library-filter]');if(reset)reset.hidden=noSaved;
    if(focused?.hidden)(document.querySelector('[data-asset-id]:not([hidden]) a')||empty.querySelector('a:not([hidden])')||document.querySelector('[data-library-query]'))?.focus();
  }
  function renderLibrary(){
    const host=document.querySelector('[data-assets]');host.replaceChildren();
    for(const asset of all()){
      const fragment=document.getElementById('asset-card').content.cloneNode(true),card=fragment.querySelector('article');card.dataset.assetId=asset.id;
      for(const link of card.querySelectorAll('[data-asset-link]'))link.href=destination(asset);
      card.querySelector('[data-asset-title]').textContent=asset.title;
      card.querySelector('[data-asset-meta]').textContent=(asset.kind==='image'?'이미지':asset.kind==='course'?'학습':'문서')+' / '+asset.date+' / '+asset.owner;
      const picture=card.querySelector('[data-asset-picture]');
      if(asset.kind==='image')picture.append(figure(asset));
      else{const cover=document.createElement('div');cover.className='asset-doc'+(asset.kind==='course'?' course':'');const label=document.createElement('small');label.textContent=asset.kind==='course'?'Archive class':'Project document';const title=document.createElement('strong');title.textContent=asset.title;const sub=document.createElement('small');sub.textContent=asset.subtitle;cover.append(title,sub);if(asset.body){const excerpt=document.createElement('p');excerpt.className='cover-excerpt';excerpt.textContent=asset.body.split('\n\n')[0];cover.append(excerpt);}if(asset.kind==='course'){const list=document.createElement('ol');list.className='cover-lessons';for(const [name]of lessons){const item=document.createElement('li');item.textContent=name;list.append(item);}cover.append(list);}cover.append(label);picture.append(cover);}
      card.querySelector('[data-save]').dataset.save=asset.id;host.append(fragment);
    }
    applyFilter();ReferenceMedia.enhance(host);ui.enter(document.querySelector('[data-library-region]'));
  }
  function renderAsset(){
    const id=new URLSearchParams(location.search).get('id')||'A-01',asset=all().find(a=>a.id===id);
    if(!asset){document.querySelector('[data-asset-view]').hidden=true;document.querySelector('[data-asset-missing]').hidden=false;return;}
    const back=document.querySelector('[data-collection-return]');if(back){const saved=new URLSearchParams(location.search).get('collection')==='saved';back.href='archive-library.html'+(saved?'?saved=1':'');back.textContent=saved?'저장한 자료로':'모든 자료로';}
    document.querySelector('[data-asset-title]').textContent=asset.title;document.querySelector('[data-asset-file]').textContent=asset.file;
    document.querySelector('[data-asset-owner]').textContent=asset.owner;document.querySelector('[data-asset-date]').textContent=asset.date;
    const viewer=document.querySelector('[data-viewer]');viewer.replaceChildren();
    document.querySelector('h1').textContent=asset.title;
    if(asset.kind==='image'){viewer.append(figure(asset));ReferenceMedia.enhance(viewer);document.querySelector('[data-asset-type]').textContent=asset.src.startsWith('data:image/png')?'PNG / 이미지':asset.src.startsWith('data:image/webp')?'WebP / 이미지':'JPEG / 이미지';}
    else{const article=document.createElement('article');article.className='prose pad';const title=document.createElement('h2');title.textContent=asset.title;article.append(title);for(const paragraph of (asset.body||asset.subtitle).split('\n\n')){const p=document.createElement('p');p.textContent=paragraph;article.append(p);}if(['guide','course'].includes(asset.kind)){const link=document.createElement('a');link.href=destination(asset);link.textContent='전체 내용 보기';article.append(link);}viewer.append(article);document.querySelector('[data-asset-type]').textContent='문서';}
    document.querySelector('[data-asset-size]').textContent=asset.size||'프로젝트 메모';
    const button=document.querySelector('[data-save]');button.hidden=false;button.dataset.save=asset.id;button.setAttribute('aria-pressed',String(state.saved.includes(asset.id)));button.querySelector('span').textContent=state.saved.includes(asset.id)?'저장됨':'자료 저장';
    const comments=document.querySelector('[data-asset-comments]');comments.replaceChildren();
    for(const message of state.comments[id]||[]){const item=document.createElement('article');const by=document.createElement('strong');by.textContent='김민서';const text=document.createElement('p');text.textContent=message;item.append(by,text);comments.append(item);}
    ui.enter(document.querySelector('main'));
  }
  function renderLesson(){
    const number=Number(new URLSearchParams(location.search).get('lesson')||2);
    if(!Number.isInteger(number)||number<1||number>4){ui.error('학습 항목을 찾을 수 없습니다.');return;}
    document.querySelector('[data-lesson-title]').textContent=lessons[number-1][0];document.querySelector('[data-lesson-copy]').textContent=lessons[number-1][1];
    document.querySelector('[data-lesson-number]').textContent=number;
    const host=document.querySelector('[data-curriculum]');host.replaceChildren();
    lessons.forEach((item,i)=>{const li=document.createElement('li');if(i+1===number)li.setAttribute('aria-current','step');const marker=document.createElement('span');marker.textContent=state.completed.includes(i+1)?'✓':String(i+1);const link=document.createElement('a');link.href='archive-lesson.html?lesson='+(i+1);link.textContent=item[0];const small=document.createElement('small');small.textContent=state.completed.includes(i+1)?'완료':i+1===number?'현재 학습':'대기';link.append(small);li.append(marker,link);host.append(li);});
    document.querySelector('[data-learning-progress]').value=state.completed.length;document.querySelector('[data-learning-count]').textContent=`4개 중 ${state.completed.length}개 완료`;
    const complete=document.querySelector('[data-complete-lesson]');complete.dataset.completeLesson=number;complete.disabled=state.completed.includes(number);complete.textContent=complete.disabled?'완료한 학습':'학습 완료로 표시';
    const next=document.querySelector('[data-next-lesson]');next.href=number<4?'archive-lesson.html?lesson='+(number+1):'archive-library.html';next.textContent=number<4?'다음 학습':'라이브러리로';ui.enter(document.querySelector('main'));
  }
  document.querySelector('[data-library-query]')?.addEventListener('input',event=>{if(!event.isComposing)applyFilter();},options);
  document.querySelector('[data-library-query]')?.addEventListener('compositionend',applyFilter,options);
  document.addEventListener('click',event=>{
    const button=event.target.closest('button');if(!button)return;
    try{
      if(button.dataset.libraryFilter){if(button.closest('[data-library-empty]'))document.querySelector('[data-library-query]').value='';const changed=filter!==button.dataset.libraryFilter;filter=button.dataset.libraryFilter;applyFilter();if(changed)ui.motion(document.querySelector('[data-assets]'),'filter');}
      if(button.dataset.save){const id=button.dataset.save;commit({...state,saved:state.saved.includes(id)?state.saved.filter(x=>x!==id):[...state.saved,id]});if(page==='library')applyFilter();else{button.setAttribute('aria-pressed',String(state.saved.includes(id)));button.querySelector('span').textContent=state.saved.includes(id)?'저장됨':'자료 저장';}ui.motion(button.querySelector('.icon'),'save');ui.toast(state.saved.includes(id)?'자료를 저장했습니다.':'저장을 해제했습니다.');}
      if(button.dataset.completeLesson){const n=Number(button.dataset.completeLesson);if(!state.completed.includes(n))commit({...state,completed:[...state.completed,n]});renderLesson();ui.toast('로컬 데모의 학습 상태를 변경했습니다.');}
    }catch(error){ui.error(error.message);}
  },options);
  const comment=document.getElementById('asset-comment');
  if(comment)comment.addEventListener('submit',event=>{event.preventDefault();try{const id=new URLSearchParams(location.search).get('id')||'A-01',text=comment.elements.comment.value.trim();if((state.comments[id]||[]).length>=20)throw new Error('자료별 댓글은 20개까지 보관할 수 있습니다.');commit({...state,comments:{...state.comments,[id]:[...(state.comments[id]||[]),text]}});comment.reset();renderAsset();ui.toast('로컬 자료에 댓글을 남겼습니다.');}catch(error){ui.error(error.message);}},options);
  const upload=document.getElementById('asset-upload');
  if(upload)upload.addEventListener('submit',async event=>{
    event.preventDefault();const file=upload.elements.file.files[0],button=upload.querySelector('[type=submit]'),title=upload.elements.title.value.trim();
    if(!file||uploadTask)return;
    const task={cancelled:false};uploadTask=task;upload.querySelector('[data-upload-error]').hidden=true;
    try{
      if(file.size>1048576||!['image/jpeg','image/png','image/webp'].includes(file.type))throw new Error('이 데모에는 1MiB 이하의 JPEG·PNG·WebP 이미지 한 개를 추가할 수 있습니다.');
      button.disabled=true;button.textContent='이미지 확인 중';upload.elements.file.disabled=true;upload.elements.title.disabled=true;
      const src=await new Promise((resolve,reject)=>{reader=new FileReader();reader.onload=()=>resolve(reader.result);reader.onerror=()=>reject(new Error('이미지를 읽지 못했습니다.'));reader.onabort=()=>reject(new DOMException('Cancelled','AbortError'));reader.readAsDataURL(file);});
      const image=new Image();image.src=src;await image.decode();if(task.cancelled||ui.signal.aborted||!upload.closest('dialog').open)return;
      if(image.naturalWidth*image.naturalHeight>4000000)throw new Error('이 로컬 데모의 이미지 해상도 한도는 400만 화소입니다.');
      const canvas=document.createElement('canvas'),scale=Math.min(40/image.naturalWidth,40/image.naturalHeight);canvas.width=Math.max(1,Math.round(image.naturalWidth*scale));canvas.height=Math.max(1,Math.round(image.naturalHeight*scale));canvas.getContext('2d').drawImage(image,0,0,canvas.width,canvas.height);
      const id='A-'+state.nextId,old=state.custom?.id,comments={...state.comments};if(old)delete comments[old];
      const asset={id,title:title||file.name,kind:'image',src,preview:canvas.toDataURL('image/jpeg',.6),file:file.name,size:Math.round(file.size/1024)+' KB',date:new Date().toLocaleDateString('ko-KR'),owner:'김민서'};
      commit({...state,custom:asset,nextId:state.nextId+1,saved:state.saved.filter(x=>x!==old),comments});filter='all';upload.closest('dialog').close();upload.reset();renderLibrary();ui.toast('이 브라우저 세션의 라이브러리에 이미지를 추가했습니다.');
    }catch(error){if(!task.cancelled&&error.name!=='AbortError'){const node=upload.querySelector('[data-upload-error]');node.textContent=error.message;node.hidden=false;}}
    finally{reader=null;uploadTask=null;button.disabled=false;button.textContent='자료 추가';upload.elements.file.disabled=false;upload.elements.title.disabled=false;}
  },options);
  const cancelUpload=()=>{if(uploadTask)uploadTask.cancelled=true;reader?.abort();};
  if(upload)upload.closest('dialog').addEventListener('close',cancelUpload,options);
  ui.onLeave(cancelUpload);
  try{state=ExampleState.read('archive',initial,valid);if(page==='library')renderLibrary();if(page==='asset')renderAsset();if(page==='lesson')renderLesson();if(page==='guide')ui.enter(document.querySelector('main'));}catch(error){ui.error(error.message);}
  window.addEventListener('pageshow',event=>{if(event.persisted){try{state=ExampleState.read('archive',initial,valid);if(page==='library')renderLibrary();if(page==='asset')renderAsset();if(page==='lesson')renderLesson();}catch(error){ui.error(error.message);}}},options);
})();
