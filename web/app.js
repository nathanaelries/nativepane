'use strict';
const $ = id => document.getElementById(id);
let session, token, revision = 0, model, mode, filename, current = new Map(), saved = new Map();
let history = [], historyIndex = 0, saveTimer, saving, errorTimer, closed = false, sheetIndex = 0, rowPage = 0, colPage = 0;
let wordLayoutTimer;
const params = new URLSearchParams(location.hash.slice(1));
const initialSession = params.get('session') || new URLSearchParams(location.search).get('session'), initialToken = params.get('token');
if (initialToken) { sessionStorage.setItem(`nativepane:${initialSession}`, initialToken); window.history.replaceState(null, '', `${location.pathname}?session=${encodeURIComponent(initialSession)}`); }
function message(text) { $('message').textContent = text; $('message').hidden = false; clearTimeout(errorTimer); errorTimer = setTimeout(() => $('message').hidden = true, 12000); }
function status(text) { $('save-status').textContent = text; }
async function request(path, options = {}, host = false) {
 const headers = new Headers(options.headers || {}); const credential = host ? $('api-key').value : token;
 if (credential) headers.set('Authorization', `Bearer ${credential}`);
 const response = await fetch(path, { ...options, headers, credentials: 'omit' });
 if (!response.ok) { let info; try { info = await response.json(); } catch {} throw new Error(info?.error || `Request failed (${response.status})`); }
 return response;
}
function endpoint(path) { return `/api/v1/sessions/${session}/${path}`; }
function notify(type) { if (parent !== window && document.referrer) { try { parent.postMessage({ source: 'nativepane', sessionId: session, type, revision }, new URL(document.referrer).origin); } catch {} } }
function dirty() { return [...current].some(([id,text]) => text !== saved.get(id)); }
function remember() {
 history = history.slice(0, historyIndex + 1); history.push(new Map(current)); if (history.length > 100) history.shift(); historyIndex = history.length - 1;
 schedule(); updateButtons();
}
function schedule() { clearTimeout(saveTimer); status(dirty() ? 'Unsaved changes…' : `Saved · revision ${revision}`); if (dirty()) saveTimer = setTimeout(() => save().catch(e => {status('Save failed · changes kept in this tab');message(e.message);}), 900); }
function updateButtons() { $('undo').disabled = mode !== 'edit' || historyIndex <= 0; $('redo').disabled = mode !== 'edit' || historyIndex >= history.length-1; $('save').disabled = mode !== 'edit'; }
function moveHistory(delta) {
 const next = historyIndex + delta; if (next < 0 || next >= history.length || mode !== 'edit') return;
 historyIndex = next; current = new Map(history[next]);
 document.querySelectorAll('[data-field]').forEach(el => { if (el instanceof HTMLInputElement) el.value = current.get(el.dataset.field); else el.textContent = current.get(el.dataset.field); });
 updateWordMirrors(); queueWordLayout();
 schedule(); updateButtons();
}
async function save() {
 clearTimeout(saveTimer); if (saving) { await saving; if (dirty()) return save(); return; }
 if (!dirty() || mode !== 'edit') return;
 const snapshot = new Map(current), edits = [...snapshot].filter(([id,text]) => text !== saved.get(id)).map(([id,text]) => ({id,text})); status('Saving…');
 saving = (async () => {const r = await request(endpoint('document'), {method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({revision,edits})}); const data = await r.json(); revision = data.revision; saved = snapshot; status(`Saved · revision ${revision}`); notify('saved');})();
 try { await saving; } finally { saving = null; }
 if (dirty()) return save();
}
function fieldElement(f, input = false) {
 const el = document.createElement(input ? 'input' : 'span'); el.dataset.field = f.id; el.setAttribute('aria-label', f.Address || f.address || `Text ${f.id}`);
 if (input) {el.value = current.get(f.id); el.readOnly = mode !== 'edit' || f.readOnly; el.className = f.kind || '';}
 else {el.className = `editable${f.bold?' bold':''}${f.italic?' italic':''}`;el.textContent = current.get(f.id);el.contentEditable = mode === 'edit' && !f.readOnly ? 'plaintext-only':'false';el.setAttribute('role','textbox');el.setAttribute('aria-multiline','true');applyWordStyle(el, f.style);}
 el.addEventListener('input', () => { current.set(f.id, input ? el.value : el.innerText.replace(/\r\n/g,'\n')); remember(); updateWordMirrors(); queueWordLayout(); });
 return el;
}
function outline(label, action) {const b = document.createElement('button');b.textContent=label;b.addEventListener('click',action);$('outline').append(b);}
const wordCSSKeys = new Set(['fontFamily','fontSize','fontWeight','fontStyle','color','textDecorationLine','letterSpacing','lineHeight','textAlign','textIndent','marginTop','marginBottom','marginLeft','marginRight','paddingTop','paddingRight','paddingBottom','paddingLeft','borderTop','borderRight','borderBottom','borderLeft','backgroundColor']);
function applyWordStyle(element, style) {
 for (const [key, value] of Object.entries(style || {})) if (wordCSSKeys.has(key)) element.style[key] = value;
}
function updateWordMirrors() {
 document.querySelectorAll('[data-mirror-field]').forEach(el => el.textContent = current.get(el.dataset.mirrorField));
}
function queueWordLayout() {
 if (model?.format !== 'docx' || $('workspace').hidden) return;
 clearTimeout(wordLayoutTimer);
 wordLayoutTimer = setTimeout(reflowWord, 350);
}
function reflowWord() {
 if (model?.format !== 'docx' || $('workspace').hidden) return;
 const active = document.activeElement;
 const id = active?.dataset.field;
 const selection = window.getSelection();
 let offset = 0;
 if (id && selection?.rangeCount && active.contains(selection.focusNode)) {
  const range = document.createRange(); range.selectNodeContents(active); range.setEnd(selection.focusNode, selection.focusOffset); offset = range.toString().length;
 }
 const scroll = document.querySelector('.canvas-wrap').scrollTop;
 renderWord();
 const field = id && [...document.querySelectorAll('[data-field]')].find(el => el.dataset.field === id);
 if (field) {
  field.focus({preventScroll:true});
  const walker = document.createTreeWalker(field, NodeFilter.SHOW_TEXT);
  let node; while ((node = walker.nextNode())) { if (offset <= node.length) { selection.setBaseAndExtent(node, offset, node, offset); break; } offset -= node.length; }
 }
 document.querySelector('.canvas-wrap').scrollTop = scroll;
}
function wordBlockElement(block) {
 if (block.kind !== 'table') {
  const paragraph = document.createElement('p');
  if (['center', 'right', 'justify'].includes(block.align)) paragraph.className = `word-align-${block.align}`;
  applyWordStyle(paragraph, block.style);
  if (block.keepNext) paragraph.dataset.keepNext = 'true';
  block.fields.forEach(f => paragraph.append(fieldElement(f)));
  if (!block.fields.length) paragraph.append(document.createElement('br'));
  return paragraph;
 }
 const table = document.createElement('table');
 table.className = 'document-table';
 applyWordStyle(table, block.style);
 table.setAttribute('aria-label', block.label || 'Document table');
 const widths = block.columnWidths || [];
 const total = widths.reduce((sum, width) => sum + width, 0);
 const cols = document.createElement('colgroup');
 widths.forEach(width => {
  const col = document.createElement('col');
  if (total > 0) col.style.width = `${width / total * 100}%`;
  cols.append(col);
 });
 table.append(cols);
 const body = document.createElement('tbody');
 // Occupancy includes vertical merges from previous rows. Only unoccupied gaps
 // get placeholder cells, preserving gridBefore/gridAfter without shifting cells.
 const occupied = new Array(widths.length).fill(0);
 for (const row of block.rows || []) {
  const tr = document.createElement('tr');
  if (row.repeatHeader) tr.dataset.repeatHeader = 'true';
  let cursor = 0;
  function fillGaps(until) {
   while (cursor < until) {
    if (occupied[cursor] > 0) { cursor++; continue; }
    const start = cursor++;
    while (cursor < until && occupied[cursor] === 0) cursor++;
    const gap = document.createElement('td');
    gap.colSpan = cursor - start;
    gap.className = 'document-table-gap';
    gap.setAttribute('aria-hidden', 'true');
    tr.append(gap);
   }
  }
  for (const cell of row.cells) {
   fillGaps(cell.column);
   const td = document.createElement('td');
   td.colSpan = cell.colSpan;
   td.rowSpan = cell.rowSpan;
   if (/^[0-9a-f]{6}$/i.test(cell.fill || '')) td.style.backgroundColor = `#${cell.fill}`;
   if (['center', 'bottom'].includes(cell.vAlign)) td.classList.add(`word-valign-${cell.vAlign}`);
   applyWordStyle(td, cell.style);
   cell.blocks.forEach(child => td.append(wordBlockElement(child)));
   tr.append(td);
   for (let col = cell.column; col < cell.column + cell.colSpan; col++) occupied[col] = cell.rowSpan;
   cursor = cell.column + cell.colSpan;
  }
  fillGaps(widths.length);
  body.append(tr);
  occupied.forEach((remaining, col) => occupied[col] = Math.max(0, remaining - 1));
 }
 table.append(body);
 return table;
}
function renderWord() {
 $('canvas').replaceChildren(); $('outline').replaceChildren(); $('sheet-nav').replaceChildren();
 const geometry = model.page || {width:12240,height:15840,top:1440,right:1440,bottom:1440,left:1440};
 const width = geometry.width / 15, height = geometry.height / 15;
 const wrap = document.querySelector('.canvas-wrap'), wrapStyle = getComputedStyle(wrap);
 const available = wrap.clientWidth - parseFloat(wrapStyle.paddingLeft) - parseFloat(wrapStyle.paddingRight);
 const scale = Math.min(1, Math.max(0.2, available / width));
 const capacity = (geometry.height - geometry.top - geometry.bottom) / 15 - 2;
 let page, content, count = 0;
 function newPage() {
  const shell = document.createElement('div'); shell.className = 'page-shell'; shell.style.width = `${width * scale}px`;
  page = document.createElement('section'); page.className = 'page';
  page.style.width = `${width}px`; page.style.minHeight = `${height}px`; page.style.padding = `${geometry.top/15}px ${geometry.right/15}px ${geometry.bottom/15}px ${geometry.left/15}px`;
  page.style.transform = `scale(${scale})`;
  content = document.createElement('div'); content.className = 'page-content'; page.append(content); shell.append(page); $('canvas').append(shell);
  count++; const target = shell; outline(`Page ${count}`, () => target.scrollIntoView({behavior:'smooth'}));
 }
 function overflowing() { return content.scrollHeight > capacity + 1; }
 function nextPageWithHeading() {
  const kept = [];
  let node = content.lastElementChild;
  while (node?.dataset.keepNext === 'true') { kept.unshift(node); node = node.previousElementSibling; }
  // An oversized keep-with-next group at the top of a page must expand that
  // page, rather than leave an empty page behind or migrate forever.
  if (kept.length && kept.length === content.children.length) return;
  kept.forEach(node => node.remove());
  newPage(); kept.forEach(node => content.append(node));
 }
 function tableFragment(source) {
  const fragment = source.cloneNode(false);
  const columns = source.querySelector(':scope > colgroup'); if (columns) fragment.append(columns.cloneNode(true));
  fragment.append(document.createElement('tbody')); return fragment;
 }
 function mirrorHeader(row) {
  const clone = row.cloneNode(true); clone.dataset.repeated = 'true';
  clone.querySelectorAll('[data-field]').forEach(el => {
   el.dataset.mirrorField = el.dataset.field; delete el.dataset.field; el.contentEditable = 'false'; el.removeAttribute('role');
  });
  return clone;
 }
 function paginateTable(block) {
  const source = wordBlockElement(block), rows = [...source.querySelector(':scope > tbody').children];
  if (!rows.length) { content.append(source); return; }
  // Groups linked by a vertical merge move together, so a page break cannot
  // silently shorten rowSpan or shift a cell into the wrong grid column.
  const groups = [];
  for (let start = 0; start < rows.length;) {
   let end = start + 1;
   for (let i = start; i < end && i < rows.length; i++) for (const cell of rows[i].children) end = Math.max(end, i + cell.rowSpan);
   end = Math.min(end, rows.length); groups.push(rows.slice(start, end)); start = end;
  }
  let headerCount = 0; while (headerCount < rows.length && rows[headerCount].dataset.repeatHeader === 'true') headerCount++;
  // A header merged into body rows cannot be repeated independently.
  const headers = rows.slice(0, headerCount);
  if (headers.some((row,i) => [...row.children].some(cell => i + cell.rowSpan > headerCount))) headers.length = 0;
  if (headerCount && headerCount < rows.length) {
   const first = []; while (groups.length && first.length <= headerCount) first.push(...groups.shift()); groups.unshift(first);
  }
  let fragment = tableFragment(source), body = fragment.querySelector(':scope > tbody'); content.append(fragment);
  for (const group of groups) {
   const already = body.children.length;
   group.forEach(row => body.append(row));
   if (overflowing() && (already > 0 || content.children.length > 1)) {
    group.forEach(row => row.remove());
    if (!already) fragment.remove();
    nextPageWithHeading();
    fragment = tableFragment(source); body = fragment.querySelector(':scope > tbody'); content.append(fragment);
    if (already > 0) headers.forEach(row => body.append(mirrorHeader(row)));
    group.forEach(row => body.append(row));
   }
  }
 }
 newPage();
 for (const block of model.blocks) {
  if (block.breakBefore && content.children.length) newPage();
  if (block.kind === 'table') { paginateTable(block); continue; }
  const element = wordBlockElement(block); content.append(element);
  if (overflowing() && content.children.length > 1) { element.remove(); nextPageWithHeading(); content.append(element); }
 }
 document.querySelectorAll('.page-shell').forEach((shell, i) => {
  const sheet = shell.firstElementChild, label = document.createElement('div'); label.className = 'page-label'; label.textContent = `${i+1} / ${count} · Approximate pagination`; sheet.append(label);
  shell.style.height = `${sheet.offsetHeight * scale}px`;
 });
 updateWordMirrors();
}
function render() {
 $('canvas').replaceChildren();$('outline').replaceChildren();$('sheet-nav').replaceChildren();
 if (model.format === 'xlsx') {renderSheet();return;}
 if (model.format === 'pptx') {
  model.blocks.forEach(b => {const slide=document.createElement('section');slide.className='slide';const title=document.createElement('h2');title.textContent=b.label;slide.append(title);b.fields.forEach(f=>{const p=document.createElement('p');p.append(fieldElement(f));slide.append(p)});$('canvas').append(slide);outline(b.label,()=>slide.scrollIntoView({behavior:'smooth'}));});return;
 }
 renderWord();
}
function columnNumber(name){return [...name].reduce((n,c)=>n*26+c.charCodeAt(0)-64,0)}
function renderSheet(){
 model.blocks.forEach((b,i)=>outline(b.label,()=>{sheetIndex=i;rowPage=colPage=0;render()}));
 const b=model.blocks[sheetIndex], byCell=new Map(b.fields.map(f=>[f.address,f]));
 const rows=[...new Set(b.fields.map(f=>Number(f.address.match(/\d+/)[0])))].sort((a,b)=>a-b);
 const cols=[...new Set(b.fields.map(f=>f.address.match(/[A-Z]+/)[0]))].sort((a,b)=>columnNumber(a)-columnNumber(b));
 function nav(label,disabled,action){const btn=document.createElement('button');btn.textContent=label;btn.disabled=disabled;btn.onclick=action;$('sheet-nav').append(btn)}
 nav('← Rows',rowPage===0,()=>{rowPage--;render()});nav(`Rows ${rowPage*100+1}–${Math.min(rows.length,(rowPage+1)*100)} / ${rows.length}`,true,()=>{});nav('Rows →',(rowPage+1)*100>=rows.length,()=>{rowPage++;render()});nav('← Columns',colPage===0,()=>{colPage--;render()});nav('Columns →',(colPage+1)*26>=cols.length,()=>{colPage++;render()});
 const table=document.createElement('table');table.className='sheet';const head=document.createElement('tr');const corner=document.createElement('th');corner.textContent=b.label;head.append(corner);const visibleCols=cols.slice(colPage*26,(colPage+1)*26);visibleCols.forEach(c=>{const th=document.createElement('th');th.textContent=c;head.append(th)});table.append(head);
 rows.slice(rowPage*100,(rowPage+1)*100).forEach(row=>{const tr=document.createElement('tr');const th=document.createElement('th');th.textContent=row;tr.append(th);visibleCols.forEach(col=>{const td=document.createElement('td'),f=byCell.get(col+row);if(f)td.append(fieldElement(f,true));tr.append(td)});table.append(tr)});$('canvas').append(table);
}
async function load(id, credential) {
 session=id;token=credential;closed=false;const r=await request(endpoint('document')),data=await r.json();({revision,model,mode,filename}=data);
 current=new Map(model.blocks.flatMap(b=>b.fields.map(f=>[f.id,f.text])));saved=new Map(current);history=[new Map(current)];historyIndex=0;sheetIndex=rowPage=colPage=0;
 $('welcome').hidden=true;$('workspace').hidden=false;$('filename').textContent=filename;$('format-badge').textContent=model.format.toUpperCase();$('mode').textContent=mode==='view'?'View only':'Text & cell editing';$('warnings').replaceChildren();model.warnings.forEach(w=>{const li=document.createElement('li');li.textContent=w;$('warnings').append(li)});
 $('document-info').textContent=`${current.size.toLocaleString()} fields · ${model.format.toUpperCase()} · Session storage`;status(`Saved · revision ${revision}`);render();updateButtons();notify('opened');
}
async function openFile(file) {
 const config=await (await request('/api/v1/config',{},true)).json();if(file.size>config.maxUploadBytes)throw new Error('File exceeds the configured upload limit.');
 const r=await request('/api/v1/sessions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({filename:file.name,mode:'edit'})},true),created=await r.json();
 try {await request(`/api/v1/sessions/${created.id}/file`,{method:'PUT',headers:{'Content-Type':'application/octet-stream'},body:file},true);}catch(e){await request(`/api/v1/sessions/${created.id}`,{method:'DELETE'},true).catch(()=>{});throw e}
 await load(created.id,created.token);
}
$('upload').addEventListener('change',e=>{if(e.target.files[0])openFile(e.target.files[0]).catch(e=>message(e.message))});
document.addEventListener('dragover',e=>e.preventDefault());document.addEventListener('drop',e=>{e.preventDefault();if(!$('welcome').hidden&&e.dataTransfer.files[0])openFile(e.dataTransfer.files[0]).catch(e=>message(e.message))});
$('url-form').addEventListener('submit',async e=>{e.preventDefault();try{const u=new URL($('source-url').value);if(!['http:','https:'].includes(u.protocol)||u.username||u.password)throw new Error('Use an HTTP(S) URL without credentials.');const config=await(await request('/api/v1/config')).json();const r=await fetch(u,{credentials:'omit',referrerPolicy:'no-referrer'});if(!r.ok||!r.body)throw new Error('Could not fetch URL. Check source CORS settings.');const reader=r.body.getReader(),chunks=[];let size=0;while(true){const {done,value}=await reader.read();if(done)break;size+=value.length;if(size>config.maxUploadBytes){await reader.cancel();throw new Error('Remote file exceeds upload limit.')}chunks.push(value)}await openFile(new File(chunks,decodeURIComponent(u.pathname.split('/').pop())))}catch(err){message(err.message)}});
$('save').onclick=()=>save().catch(e=>{status('Save failed · changes kept in this tab');message(e.message)});
$('undo').onclick=()=>moveHistory(-1);$('redo').onclick=()=>moveHistory(1);
$('download').onclick=async()=>{try{await save();const r=await request(endpoint('file')),blob=await r.blob(),url=URL.createObjectURL(blob),a=document.createElement('a');a.href=url;a.download=filename;a.click();setTimeout(()=>URL.revokeObjectURL(url),30000);notify('downloaded')}catch(e){message(e.message)}};
$('close').onclick=async()=>{try{await save();await request(endpoint('events'),{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({type:'closed'})});closed=true;notify('closed');sessionStorage.removeItem(`nativepane:${session}`);$('workspace').hidden=true;$('welcome').hidden=false;current.clear();saved.clear();session=null;token=null}catch(e){message(e.message)}};
window.addEventListener('beforeunload',e=>{if(dirty()||saving){e.preventDefault();e.returnValue=''}});
window.addEventListener('pagehide',()=>{if(session&&!closed)fetch(endpoint('events'),{method:'POST',headers:{'Content-Type':'application/json',Authorization:`Bearer ${token}`},body:JSON.stringify({type:'closed'}),keepalive:true}).catch(()=>{})});
document.addEventListener('keydown',e=>{if(!session||!(e.ctrlKey||e.metaKey))return;if(e.key.toLowerCase()==='z'){e.preventDefault();moveHistory(e.shiftKey?1:-1)}if(e.key.toLowerCase()==='s'){e.preventDefault();save().catch(e=>message(e.message))}});
window.addEventListener('resize', queueWordLayout);
(async()=>{try{const config=await(await request('/api/v1/config')).json();$('auth-panel').hidden=config.auth==='none';if(initialSession)await load(initialSession,initialToken||sessionStorage.getItem(`nativepane:${initialSession}`));else if(location.pathname==='/embed')message('Open this pane using the signed embed URL returned by the session API.')}catch(e){message(e.message)}})();
