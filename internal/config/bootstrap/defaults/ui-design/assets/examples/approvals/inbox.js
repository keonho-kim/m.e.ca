// Purpose: Filter the one static request table by its visible review state.
// Pattern: Page-owned DOM projection with native radio semantics.
// Usage: approvals/inbox.html loads this after shared/interaction.js.
// Related: internal/config/bootstrap/defaults/ui-design/assets/examples/approvals/approvals.css.
(() => {
  const filters=document.querySelector('.approval-filters');
  const table=document.getElementById('request-list');
  const rows=[...table.querySelectorAll('tbody tr')];
  const count=document.querySelector('[data-filter-count]');
  const empty=document.querySelector('[data-filter-empty]');
  filters.addEventListener('change',event=>{
    const selected=filters.querySelector(':checked').value;
    let shown=0;
    for(const row of rows){
      const matches=selected==='all'||row.dataset.state===selected||(selected==='review'&&row.dataset.state==='waiting');
      row.hidden=!matches;if(matches)shown++;
    }
    count.textContent=`요청 문서 ${shown}개`;empty.hidden=shown!==0;
    if(event.isTrusted)ExampleUI.motion(table,'filter');
  },{signal:ExampleUI.signal});
})();
