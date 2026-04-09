package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Dossier</title>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--blue:#5b8dd9;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',serif}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--serif);line-height:1.6}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap}
.hdr h1{font-family:var(--mono);font-size:.9rem;letter-spacing:2px}
.hdr h1 span{color:var(--rust)}
.main{padding:1.5rem;max-width:960px;margin:0 auto}
.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:.5rem;margin-bottom:1rem}
.st{background:var(--bg2);border:1px solid var(--bg3);padding:.6rem;text-align:center;font-family:var(--mono)}
.st-v{font-size:1.3rem;font-weight:700}
.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.15rem}
.toolbar{display:flex;gap:.5rem;margin-bottom:1rem;align-items:center;flex-wrap:wrap}
.search{flex:1;min-width:180px;padding:.4rem .6rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.search:focus{outline:none;border-color:var(--leather)}
.filter-sel{padding:.4rem .5rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.65rem}
.cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:.5rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;transition:border-color .2s}
.card:hover{border-color:var(--leather)}
.card-top{display:flex;justify-content:space-between;align-items:flex-start;gap:.5rem}
.card-name{font-size:.9rem;font-weight:700}
.card-role{font-size:.72rem;color:var(--cd);margin-top:.1rem}
.card-contact{font-family:var(--mono);font-size:.6rem;margin-top:.4rem;display:flex;flex-direction:column;gap:.15rem}
.card-contact a{color:var(--blue);text-decoration:none}
.card-contact a:hover{color:var(--rust)}
.card-meta{font-family:var(--mono);font-size:.55rem;color:var(--cm);margin-top:.35rem;display:flex;gap:.4rem;flex-wrap:wrap;align-items:center}
.card-notes{font-size:.7rem;color:var(--cm);margin-top:.35rem;font-style:italic;padding:.3rem .5rem;border-left:2px solid var(--bg3)}
.card-extra{font-family:var(--mono);font-size:.58rem;color:var(--cd);margin-top:.35rem;display:flex;flex-direction:column;gap:.15rem;padding-top:.35rem;border-top:1px dashed var(--bg3)}
.card-extra-row{display:flex;gap:.4rem}
.card-extra-label{color:var(--cm);text-transform:uppercase;letter-spacing:.5px;min-width:90px}
.card-extra-val{color:var(--cream)}
.tag{font-family:var(--mono);font-size:.5rem;padding:.1rem .3rem;background:var(--bg3);color:var(--cd)}
.badge{font-family:var(--mono);font-size:.5rem;padding:.12rem .35rem;text-transform:uppercase;letter-spacing:1px;border:1px solid}
.badge.active{border-color:var(--green);color:var(--green)}
.badge.lead{border-color:var(--gold);color:var(--gold)}
.badge.inactive{border-color:var(--cm);color:var(--cm)}
.badge.client{border-color:var(--blue);color:var(--blue)}
.card-actions{display:flex;gap:.3rem;flex-shrink:0}
.btn{font-family:var(--mono);font-size:.6rem;padding:.25rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .2s}
.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}
.btn-sm{font-size:.55rem;padding:.2rem .4rem}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}
.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:480px;max-width:92vw;max-height:90vh;overflow-y:auto}
.modal h2{font-family:var(--mono);font-size:.8rem;margin-bottom:1rem;color:var(--rust);letter-spacing:1px}
.fr{margin-bottom:.6rem}
.fr label{display:block;font-family:var(--mono);font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.fr input:focus,.fr select:focus,.fr textarea:focus{outline:none;border-color:var(--leather)}
.fr-section{margin-top:1rem;padding-top:.8rem;border-top:1px solid var(--bg3)}
.fr-section-label{font-family:var(--mono);font-size:.55rem;color:var(--rust);text-transform:uppercase;letter-spacing:1px;margin-bottom:.5rem}
.row2{display:grid;grid-template-columns:1fr 1fr;gap:.5rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.85rem}
.count-label{font-family:var(--mono);font-size:.6rem;color:var(--cm);margin-bottom:.5rem}
@media(max-width:600px){.cards{grid-template-columns:1fr}.row2{grid-template-columns:1fr}.toolbar{flex-direction:column}.search{min-width:100%}}

/* Import flow */
.modal-wide{width:720px}
.import-drop{border:2px dashed var(--bg3);padding:2rem 1.5rem;text-align:center;cursor:pointer;transition:border-color .2s,background .2s;font-family:var(--mono);font-size:.75rem;color:var(--cd)}
.import-drop:hover,.import-drop.dragging{border-color:var(--rust);background:var(--bg)}
.import-drop strong{display:block;font-size:.85rem;color:var(--cream);margin-bottom:.4rem}
.import-drop em{display:block;font-size:.6rem;color:var(--cm);margin-top:.6rem;font-style:normal;letter-spacing:.5px;text-transform:uppercase}
.import-drop input[type=file]{display:none}
.import-tip{font-size:.65rem;color:var(--cm);margin-top:.8rem;line-height:1.5;font-style:italic}
.import-table{width:100%;border-collapse:collapse;margin-top:.8rem;font-family:var(--mono);font-size:.6rem}
.import-table th,.import-table td{padding:.4rem .5rem;border:1px solid var(--bg3);text-align:left;vertical-align:top;max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.import-table th{background:var(--bg);color:var(--cm);text-transform:uppercase;font-size:.5rem;letter-spacing:1px}
.import-table td{color:var(--cd)}
.import-table select{width:100%;padding:.25rem .3rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.55rem}
.import-table select:focus{outline:none;border-color:var(--leather)}
.import-summary{font-family:var(--mono);font-size:.7rem;color:var(--cd);margin:.8rem 0;padding:.6rem .8rem;background:var(--bg);border-left:3px solid var(--leather)}
.import-summary strong{color:var(--cream)}
.import-warning{font-family:var(--mono);font-size:.6rem;color:var(--gold);margin-top:.4rem;padding:.4rem .6rem;border-left:2px solid var(--gold);background:var(--bg)}
.import-result{font-family:var(--mono);font-size:.7rem;line-height:1.8;padding:1rem;background:var(--bg);border-left:3px solid var(--green)}
.import-result strong{color:var(--green);font-size:1rem;display:block;margin-bottom:.4rem}
.import-result .row{display:flex;justify-content:space-between;color:var(--cd)}
.import-result .row span:last-child{color:var(--cream);font-weight:700}
.import-error{font-family:var(--mono);font-size:.65rem;color:var(--red);padding:.8rem;background:var(--bg);border-left:3px solid var(--red);margin-top:.5rem;line-height:1.6}
.import-table-wrap{max-height:300px;overflow:auto;border:1px solid var(--bg3);margin-top:.5rem}

/* Trial banner */
.trial-bar{display:none;background:linear-gradient(90deg,#3a2419,#2e1c14);border-bottom:2px solid var(--rust);padding:.7rem 1.5rem;font-family:var(--mono);font-size:.68rem;color:var(--cream);align-items:center;gap:1rem;flex-wrap:wrap}
.trial-bar.show{display:flex}
.trial-bar-msg{flex:1;min-width:240px;line-height:1.5}
.trial-bar-msg strong{color:var(--rust);text-transform:uppercase;letter-spacing:1px;font-size:.6rem;display:block;margin-bottom:.15rem}
.trial-bar-actions{display:flex;gap:.5rem;align-items:center;flex-wrap:wrap}
.trial-bar a.btn-trial{background:var(--rust);color:#fff;padding:.4rem .8rem;text-decoration:none;font-size:.65rem;text-transform:uppercase;letter-spacing:1px;font-weight:700;border:1px solid var(--rust);transition:all .2s}
.trial-bar a.btn-trial:hover{background:#f08545;border-color:#f08545}
.trial-bar-divider{color:var(--cm);font-size:.6rem}
.trial-bar input.key-input{padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.6rem;width:200px}
.trial-bar input.key-input:focus{outline:none;border-color:var(--rust)}
.trial-bar button.btn-activate{padding:.4rem .7rem;background:var(--bg2);color:var(--cream);border:1px solid var(--leather);font-family:var(--mono);font-size:.6rem;cursor:pointer;text-transform:uppercase;letter-spacing:1px}
.trial-bar button.btn-activate:hover{background:var(--bg3)}
.trial-bar button.btn-activate:disabled{opacity:.5;cursor:wait}
.trial-msg{font-size:.6rem;color:var(--cm);margin-left:.5rem}
.trial-msg.error{color:var(--red)}
.trial-msg.success{color:var(--green)}
.btn-disabled-trial{opacity:.4;cursor:not-allowed!important;position:relative}
.btn-disabled-trial:hover{border-color:var(--bg3)!important;color:var(--cd)!important}
@media(max-width:600px){.trial-bar{flex-direction:column;align-items:stretch}.trial-bar input.key-input{width:100%}}
</style>
</head>
<body>

<div class="trial-bar" id="trial-bar">
<div class="trial-bar-msg">
<strong>Trial Required</strong>
You can view your existing data, but adding, editing, importing, or deleting records is locked until you start a 14-day free trial.
</div>
<div class="trial-bar-actions">
<a class="btn-trial" href="https://stockyard.dev/" target="_blank" rel="noopener">Start 14-Day Trial</a>
<span class="trial-bar-divider">or</span>
<input type="text" class="key-input" id="trial-key-input" placeholder="SY-..." autocomplete="off" spellcheck="false">
<button class="btn-activate" id="trial-activate-btn" onclick="activateLicense()">Activate</button>
<span class="trial-msg" id="trial-msg"></span>
</div>
</div>

<div class="hdr">
<h1 id="dash-title"><span>&#9670;</span> DOSSIER</h1>
<div style="display:flex;gap:.5rem;flex-wrap:wrap">
<button class="btn" onclick="openImport()">Import CSV</button>
<button class="btn btn-p" onclick="openForm()">+ Add Contact</button>
</div>
</div>

<div class="main">
<div class="stats" id="stats"></div>
<div class="toolbar">
<input class="search" id="search" placeholder="Search name, email, company, tags..." oninput="render()">
<select class="filter-sel" id="status-filter" onchange="render()">
<option value="">All Status</option>
<option value="active">Active</option>
<option value="lead">Lead</option>
<option value="client">Client</option>
<option value="inactive">Inactive</option>
</select>
</div>
<div class="count-label" id="count"></div>
<div class="cards" id="contacts"></div>
</div>

<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()">
<div class="modal" id="mdl"></div>
</div>

<script>
// API base
var A='/api';
// The single resource this tool manages.
var RESOURCE='contacts';

// Field defs drive the form, the cards, and the submit body.
// Each field has: name (data key), label, type, options (for select),
// required (bool), and optional placeholder.
// Custom fields injected from /api/config get isCustom=true and are
// stored in the extras table instead of the main contacts table.
var fields=[
{name:'name',label:'Name',type:'text',required:true,placeholder:'Full name'},
{name:'email',label:'Email',type:'email'},
{name:'phone',label:'Phone',type:'tel'},
{name:'company',label:'Company',type:'text'},
{name:'title',label:'Title / Role',type:'text'},
{name:'status',label:'Status',type:'select',options:['active','lead','client','inactive']},
{name:'tags',label:'Tags',type:'text',placeholder:'comma separated'},
{name:'notes',label:'Notes',type:'textarea'}
];

var contacts=[],editId=null;

// ─── Loading and rendering ────────────────────────────────────────

async function load(){
try{
var resp=await fetch(A+'/'+RESOURCE).then(function(r){return r.json()});
var items=resp[RESOURCE]||[];
// Fetch extras and merge into each contact under matching field names.
try{
var extras=await fetch(A+'/extras/'+RESOURCE).then(function(r){return r.json()});
items.forEach(function(c){
var ex=extras[c.id];
if(!ex)return;
Object.keys(ex).forEach(function(k){if(c[k]===undefined)c[k]=ex[k]});
});
}catch(e){/* extras endpoint may not exist on old builds */}
contacts=items;
}catch(e){
console.error('load failed',e);
contacts=[];
}
renderStats();
render();
}

function renderStats(){
var total=contacts.length;
var companies={};
contacts.forEach(function(c){if(c.company)companies[c.company]=true});
var withEmail=contacts.filter(function(c){return c.email}).length;
document.getElementById('stats').innerHTML=[
{l:'Contacts',v:total},
{l:'Companies',v:Object.keys(companies).length},
{l:'With Email',v:withEmail}
].map(function(x){return '<div class="st"><div class="st-v">'+x.v+'</div><div class="st-l">'+x.l+'</div></div>'}).join('');
}

function render(){
var q=(document.getElementById('search').value||'').toLowerCase();
var sf=document.getElementById('status-filter').value;
var f=contacts;
if(sf)f=f.filter(function(c){return c.status===sf});
if(q)f=f.filter(function(c){
if((c.name||'').toLowerCase().includes(q))return true;
if((c.email||'').toLowerCase().includes(q))return true;
if((c.company||'').toLowerCase().includes(q))return true;
if((c.tags||'').toLowerCase().includes(q))return true;
return false;
});
document.getElementById('count').textContent=f.length+' contact'+(f.length!==1?'s':'');
if(!f.length){
var msg=window._emptyMsg||'No contacts found.';
document.getElementById('contacts').innerHTML='<div class="empty">'+esc(msg)+'</div>';
return;
}
var h='';
f.forEach(function(c){
h+=cardHTML(c);
});
document.getElementById('contacts').innerHTML=h;
}

function cardHTML(c){
var h='<div class="card"><div class="card-top"><div>';
h+='<div class="card-name">'+esc(c.name)+'</div>';
if(c.company||c.title){
h+='<div class="card-role">';
if(c.title)h+=esc(c.title);
if(c.title&&c.company)h+=' at ';
if(c.company)h+=esc(c.company);
h+='</div>';
}
h+='</div><div class="card-actions">';
if(!window._trialRequired){
h+='<button class="btn btn-sm" onclick="openEdit(\''+c.id+'\')">Edit</button>';
h+='<button class="btn btn-sm" onclick="del(\''+c.id+'\')" style="color:var(--red)">&#10005;</button>';
}
h+='</div></div>';
h+='<div class="card-contact">';
if(c.email)h+='<a href="mailto:'+esc(c.email)+'">'+esc(c.email)+'</a>';
if(c.phone)h+='<span>'+esc(c.phone)+'</span>';
h+='</div>';
h+='<div class="card-meta">';
if(c.status)h+='<span class="badge '+esc(c.status)+'">'+esc(c.status)+'</span>';
if(c.tags){
c.tags.split(',').forEach(function(t){
t=t.trim();
if(t)h+='<span class="tag">#'+esc(t)+'</span>';
});
}
h+='</div>';
if(c.notes)h+='<div class="card-notes">'+esc(c.notes)+'</div>';

// Custom fields from personalization render in their own block at the bottom
// of each card so they're visible at-a-glance, not just in the edit form.
var customRows='';
fields.forEach(function(f){
if(!f.isCustom)return;
var v=c[f.name];
if(v===undefined||v===null||v==='')return;
customRows+='<div class="card-extra-row">';
customRows+='<span class="card-extra-label">'+esc(f.label)+'</span>';
customRows+='<span class="card-extra-val">'+esc(String(v))+'</span>';
customRows+='</div>';
});
if(customRows)h+='<div class="card-extra">'+customRows+'</div>';

h+='</div>';
return h;
}

// ─── Form ─────────────────────────────────────────────────────────

function formHTML(contact){
var i=contact||{};
var isEdit=!!contact;
var h='<h2>'+(isEdit?'EDIT CONTACT':'NEW CONTACT')+'</h2>';

// Native fields, two per row where they pair naturally.
var pairs=[['email','phone'],['company','title'],['status','tags']];
var paired={};
pairs.forEach(function(p){paired[p[0]]=p[1];paired[p[1]]=p[0]});
var rendered={};

fields.forEach(function(f){
if(f.isCustom)return; // custom fields render in their own section below
if(rendered[f.name])return;
var partner=paired[f.name];
if(partner&&!rendered[partner]){
var pf=fieldByName(partner);
if(pf&&!pf.isCustom){
h+='<div class="row2">';
h+=fieldHTML(f,i[f.name]);
h+=fieldHTML(pf,i[partner]);
h+='</div>';
rendered[f.name]=true;
rendered[partner]=true;
return;
}
}
h+=fieldHTML(f,i[f.name]);
rendered[f.name]=true;
});

// Custom fields injected by personalization get their own labeled section.
var customFields=fields.filter(function(f){return f.isCustom});
if(customFields.length){
var sectionLabel=window._customSectionLabel||'Custom Details';
h+='<div class="fr-section"><div class="fr-section-label">'+esc(sectionLabel)+'</div>';
customFields.forEach(function(f){
h+=fieldHTML(f,i[f.name]);
});
h+='</div>';
}

h+='<div class="acts">';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='<button class="btn btn-p" onclick="submit()">'+(isEdit?'Save':'Add Contact')+'</button>';
h+='</div>';
return h;
}

function fieldByName(n){
for(var i=0;i<fields.length;i++)if(fields[i].name===n)return fields[i];
return null;
}

function fieldHTML(f,value){
var v=value;
if(v===undefined||v===null)v='';
var req=f.required?' *':'';
var ph='';
if(f.placeholder)ph=' placeholder="'+esc(f.placeholder)+'"';
else if(f.name==='name'&&window._placeholderName)ph=' placeholder="'+esc(window._placeholderName)+'"';

var h='<div class="fr"><label>'+esc(f.label)+req+'</label>';

if(f.type==='select'){
h+='<select id="f-'+f.name+'">';
if(!f.required)h+='<option value="">Select...</option>';
(f.options||[]).forEach(function(o){
var sel=(String(v)===String(o))?' selected':'';
var disp=(typeof o==='string')?(o.charAt(0).toUpperCase()+o.slice(1)):String(o);
h+='<option value="'+esc(String(o))+'"'+sel+'>'+esc(disp)+'</option>';
});
h+='</select>';
}else if(f.type==='textarea'){
h+='<textarea id="f-'+f.name+'" rows="3"'+ph+'>'+esc(String(v))+'</textarea>';
}else if(f.type==='checkbox'){
h+='<input type="checkbox" id="f-'+f.name+'"'+(v?' checked':'')+' style="width:auto">';
}else{
var inputType=f.type||'text';
if(inputType==='phone')inputType='tel';
h+='<input type="'+esc(inputType)+'" id="f-'+f.name+'" value="'+esc(String(v))+'"'+ph+'>';
}

h+='</div>';
return h;
}

function openForm(){
editId=null;
document.getElementById('mdl').innerHTML=formHTML();
document.getElementById('mbg').classList.add('open');
var nameEl=document.getElementById('f-name');
if(nameEl)nameEl.focus();
}

function openEdit(id){
var c=null;
for(var j=0;j<contacts.length;j++){if(contacts[j].id===id){c=contacts[j];break}}
if(!c)return;
editId=id;
document.getElementById('mdl').innerHTML=formHTML(c);
document.getElementById('mbg').classList.add('open');
}

function closeModal(){
document.getElementById('mbg').classList.remove('open');
editId=null;
}

// ─── Submit (split native + custom) ───────────────────────────────

async function submit(){
var nameEl=document.getElementById('f-name');
if(!nameEl||!nameEl.value.trim()){alert('Name is required');return}

var body={};
var extras={};
fields.forEach(function(f){
var el=document.getElementById('f-'+f.name);
if(!el)return;
var val;
if(f.type==='checkbox')val=el.checked;
else if(f.type==='number')val=parseFloat(el.value)||0;
else val=el.value.trim();
if(f.isCustom)extras[f.name]=val;
else body[f.name]=val;
});

var savedId=editId;
try{
if(editId){
var r1=await fetch(A+'/'+RESOURCE+'/'+editId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r1.ok){var e1=await r1.json().catch(function(){return{}});alert(e1.error||'Save failed');return}
}else{
var r2=await fetch(A+'/'+RESOURCE,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r2.ok){var e2=await r2.json().catch(function(){return{}});alert(e2.error||'Save failed');return}
var created=await r2.json();
savedId=created.id;
}
// Save extras separately if any custom fields had values
if(savedId&&Object.keys(extras).length){
await fetch(A+'/extras/'+RESOURCE+'/'+savedId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(extras)}).catch(function(){});
}
}catch(e){
alert('Network error: '+e.message);
return;
}

closeModal();
load();
}

async function del(id){
if(!confirm('Remove this contact?'))return;
await fetch(A+'/'+RESOURCE+'/'+id,{method:'DELETE'});
load();
}

function esc(s){
if(s===undefined||s===null)return'';
var d=document.createElement('div');
d.textContent=String(s);
return d.innerHTML;
}

document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal()});

// ─── CSV Import ───────────────────────────────────────────────────
//
// Three-state modal flow:
//   1. Drop zone — user picks/drops a CSV file
//   2. Mapping UI — auto-detected column → field map, user can adjust
//   3. Result — created/skipped/failed counts after commit
//
// State is held in window._import so closing and reopening the modal
// resets cleanly. We don't bother with a state machine because each
// transition is a single function call that rewrites the modal HTML.

window._import=null;

function openImport(){
window._import={csv:'',headers:[],sample:[],mapping:{},fields:[],totalRows:0,warnings:[]};
renderImportStep1();
document.getElementById('mbg').classList.add('open');
}

// Step 1: drop zone
function renderImportStep1(){
var h='<h2>IMPORT CONTACTS FROM CSV</h2>';
h+='<div class="import-drop" id="drop" onclick="document.getElementById(\'csv-file\').click()">';
h+='<strong>Drop a CSV file here</strong>';
h+='or click to browse';
h+='<em>Common sources: Mailchimp, HubSpot, Google Contacts, any spreadsheet exported as CSV</em>';
h+='<input type="file" id="csv-file" accept=".csv,text/csv" onchange="onCSVFileChosen(this)">';
h+='</div>';
h+='<div class="import-tip">We will show you which columns we detected before anything is imported. Duplicates are skipped automatically (matched on name + email + phone).</div>';
h+='<div class="acts"><button class="btn" onclick="closeModal()">Cancel</button></div>';
var mdl=document.getElementById('mdl');
mdl.className='modal';
mdl.innerHTML=h;

// Wire up drag-and-drop on the drop zone after the HTML is in place
var dz=document.getElementById('drop');
dz.addEventListener('dragover',function(e){e.preventDefault();dz.classList.add('dragging')});
dz.addEventListener('dragleave',function(){dz.classList.remove('dragging')});
dz.addEventListener('drop',function(e){
e.preventDefault();
dz.classList.remove('dragging');
if(e.dataTransfer.files.length){onCSVFileRead(e.dataTransfer.files[0])}
});
}

function onCSVFileChosen(input){
if(input.files&&input.files[0])onCSVFileRead(input.files[0]);
}

function onCSVFileRead(file){
if(!file)return;
// Refuse anything obviously not a CSV based on size and extension.
// 5MB matches the server cap; bigger files get rejected client-side
// so the user doesn't wait for an upload that will fail.
if(file.size>5*1024*1024){
alert('That CSV is larger than 5MB. The current cap is about 50,000 contacts. If you need more, email hello@stockyard.dev.');
return;
}
var reader=new FileReader();
reader.onload=function(e){
window._import.csv=e.target.result;
runImportPreview();
};
reader.onerror=function(){alert('Could not read the file. Try again, or email hello@stockyard.dev.')};
reader.readAsText(file);
}

// Step 2: call /api/import/preview, show mapping UI
async function runImportPreview(){
renderImportLoading('Reading your CSV...');
try{
var resp=await fetch(A+'/import/preview',{
method:'POST',
headers:{'Content-Type':'application/json'},
body:JSON.stringify({csv:window._import.csv})
});
var data=await resp.json();
if(!resp.ok){
renderImportError(data.error||('Preview failed: HTTP '+resp.status));
return;
}
window._import.headers=data.headers||[];
window._import.sample=data.sample_rows||[];
window._import.mapping=data.mapping||{};
window._import.fields=data.fields||[];
window._import.totalRows=data.total_rows||0;
window._import.warnings=data.warnings||[];
renderImportStep2();
}catch(e){
renderImportError('Could not reach the server: '+e.message);
}
}

function renderImportStep2(){
var d=window._import;
var h='<h2>REVIEW COLUMNS</h2>';
h+='<div class="import-summary">';
h+='Found <strong>'+d.totalRows+'</strong> row'+(d.totalRows===1?'':'s')+' across <strong>'+d.headers.length+'</strong> column'+(d.headers.length===1?'':'s')+'. ';
h+='Match each column to a contact field. Columns set to <em>Skip</em> are ignored. ';
h+='At least one column must be mapped to <strong>Name</strong>.';
h+='</div>';
if(d.warnings.length){
d.warnings.forEach(function(w){h+='<div class="import-warning">'+esc(w)+'</div>'});
}

// Mapping table — header row shows headers + dropdowns, body shows sample rows
h+='<div class="import-table-wrap"><table class="import-table">';
h+='<thead><tr>';
d.headers.forEach(function(header,i){
h+='<th>'+esc(header)+'<br><select id="map-'+i+'" onchange="onMappingChange('+i+')">';
d.fields.forEach(function(f){
var sel=(d.mapping[header]===f.key)?' selected':'';
h+='<option value="'+esc(f.key)+'"'+sel+'>'+esc(f.label)+'</option>';
});
h+='</select></th>';
});
h+='</tr></thead>';
h+='<tbody>';
d.sample.forEach(function(row){
h+='<tr>';
d.headers.forEach(function(_,i){
var v=(i<row.length)?row[i]:'';
h+='<td title="'+esc(v)+'">'+esc(v)+'</td>';
});
h+='</tr>';
});
h+='</tbody></table></div>';

h+='<div class="acts">';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='<button class="btn btn-p" id="import-go" onclick="runImportCommit()">Import '+d.totalRows+' Row'+(d.totalRows===1?'':'s')+'</button>';
h+='</div>';

var mdl=document.getElementById('mdl');
mdl.className='modal modal-wide';
mdl.innerHTML=h;
updateImportButtonState();
}

function onMappingChange(headerIdx){
var d=window._import;
var header=d.headers[headerIdx];
var newField=document.getElementById('map-'+headerIdx).value;
d.mapping[header]=newField;
updateImportButtonState();
}

function updateImportButtonState(){
// Disable the Import button if no column is mapped to Name.
var d=window._import;
var hasName=false;
for(var k in d.mapping){if(d.mapping[k]==='name'){hasName=true;break}}
var btn=document.getElementById('import-go');
if(btn){
btn.disabled=!hasName;
btn.style.opacity=hasName?'1':'.4';
btn.style.cursor=hasName?'pointer':'not-allowed';
btn.title=hasName?'':'Map at least one column to Name';
}
}

// Step 3: call /api/import/commit, show result
async function runImportCommit(){
var d=window._import;
// Final client-side guard — server validates this too but the
// button shouldn't have been clickable in the first place.
var hasName=false;
for(var k in d.mapping){if(d.mapping[k]==='name'){hasName=true;break}}
if(!hasName){alert('Map at least one column to Name first.');return}

renderImportLoading('Importing '+d.totalRows+' row'+(d.totalRows===1?'':'s')+'...');
try{
var resp=await fetch(A+'/import/commit',{
method:'POST',
headers:{'Content-Type':'application/json'},
body:JSON.stringify({csv:d.csv,mapping:d.mapping})
});
var data=await resp.json();
if(!resp.ok){
renderImportError(data.error||('Import failed: HTTP '+resp.status));
return;
}
renderImportStep3(data);
load(); // refresh the contact list in the background
}catch(e){
renderImportError('Could not reach the server: '+e.message);
}
}

function renderImportStep3(result){
var h='<h2>IMPORT COMPLETE</h2>';
h+='<div class="import-result">';
h+='<strong>'+result.created+' contact'+(result.created===1?'':'s')+' added</strong>';
h+='<div class="row"><span>New contacts created</span><span>'+result.created+'</span></div>';
h+='<div class="row"><span>Duplicates skipped</span><span>'+result.skipped+'</span></div>';
if(result.failed>0){
h+='<div class="row"><span>Rows that failed</span><span style="color:var(--red)">'+result.failed+'</span></div>';
}
h+='</div>';

if(result.errors&&result.errors.length){
h+='<div class="import-error"><strong style="display:block;margin-bottom:.4rem">Failed rows:</strong>';
result.errors.slice(0,10).forEach(function(err){
h+='&bull; '+esc(err)+'<br>';
});
if(result.errors.length>10){
h+='<em>...and '+(result.errors.length-10)+' more</em>';
}
h+='</div>';
}

h+='<div class="acts" style="margin-top:1.2rem">';
h+='<button class="btn btn-p" onclick="closeModal()">Done</button>';
h+='</div>';

var mdl=document.getElementById('mdl');
mdl.className='modal';
mdl.innerHTML=h;
}

function renderImportLoading(msg){
var h='<h2>IMPORTING</h2>';
h+='<div style="text-align:center;padding:2rem;font-family:var(--mono);font-size:.75rem;color:var(--cd)">';
h+=esc(msg);
h+='</div>';
var mdl=document.getElementById('mdl');
mdl.className='modal';
mdl.innerHTML=h;
}

function renderImportError(msg){
var h='<h2>IMPORT ERROR</h2>';
h+='<div class="import-error">'+esc(msg)+'</div>';
h+='<div class="acts" style="margin-top:1rem">';
h+='<button class="btn" onclick="renderImportStep1()">Try Again</button>';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='</div>';
var mdl=document.getElementById('mdl');
mdl.className='modal';
mdl.innerHTML=h;
}

// ─── Personalization: load /api/config and inject overrides ───────

(function loadPersonalization(){
fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
if(!cfg||typeof cfg!=='object')return;

// Override the page title and dashboard heading
if(cfg.dashboard_title){
var h1=document.getElementById('dash-title');
if(h1)h1.innerHTML='<span>&#9670;</span> '+esc(cfg.dashboard_title);
document.title=cfg.dashboard_title;
}

// Empty state message and placeholder name are read by render()/fieldHTML()
if(cfg.empty_state_message)window._emptyMsg=cfg.empty_state_message;
if(cfg.placeholder_name)window._placeholderName=cfg.placeholder_name;

// Section label for custom fields (e.g. "EMDR Details")
if(cfg.primary_label)window._customSectionLabel=cfg.primary_label+' Details';

// Inject custom fields into the field defs
if(Array.isArray(cfg.custom_fields)){
cfg.custom_fields.forEach(function(cf){
if(!cf||!cf.name||!cf.label)return;
// Don't shadow native fields
if(fieldByName(cf.name))return;
fields.push({
name:cf.name,
label:cf.label,
type:cf.type||'text',
options:cf.options||[],
isCustom:true
});
});
}
}).catch(function(){
// Config endpoint missing or unreachable — that's fine, use defaults
}).finally(function(){
load();
checkTrialState();
});
})();

// ─── Trial-required state ─────────────────────────────────────────
//
// Calls /api/tier on dashboard load. If the tool is in trial-required
// state, shows the top banner and disables write controls. The banner
// has an inline "paste your key" input so customers who already paid
// can recover from a lost env var without leaving the dashboard.
//
// We track trial state on window._trialRequired so other UI code (like
// the rendering loop in render()) can avoid showing edit/delete buttons.
window._trialRequired=false;

async function checkTrialState(){
try{
var resp=await fetch(A+'/tier');
if(!resp.ok)return;
var data=await resp.json();
window._trialRequired=!!data.trial_required;
if(window._trialRequired){
document.getElementById('trial-bar').classList.add('show');
disableWriteControls();
// Re-render so the cards stop showing edit/delete buttons
if(typeof render==='function')render();
}else{
document.getElementById('trial-bar').classList.remove('show');
}
}catch(e){
// /api/tier not reachable — assume the tool is broken in some other
// way and let the regular load() error surface that. Don't block.
}
}

function disableWriteControls(){
// Find every "+ Add Contact" / "Import CSV" button in the header and
// neutralize it. We don't remove the buttons because the user needs
// to know they exist (and that activating a license unlocks them).
var buttons=document.querySelectorAll('.hdr .btn, .hdr .btn-p');
buttons.forEach(function(b){
if(b.textContent.indexOf('Add Contact')!==-1||b.textContent.indexOf('Import')!==-1){
b.classList.add('btn-disabled-trial');
b.title='Locked: trial required';
b.setAttribute('data-original-onclick',b.getAttribute('onclick')||'');
b.onclick=function(e){
e.preventDefault();
showTrialNudge();
return false;
};
}
});
}

function showTrialNudge(){
var input=document.getElementById('trial-key-input');
if(input){
input.focus();
input.style.borderColor='var(--rust)';
setTimeout(function(){if(input)input.style.borderColor=''},1500);
}
}

async function activateLicense(){
var input=document.getElementById('trial-key-input');
var btn=document.getElementById('trial-activate-btn');
var msg=document.getElementById('trial-msg');
if(!input||!btn||!msg)return;
var key=(input.value||'').trim();
if(!key){
msg.className='trial-msg error';
msg.textContent='Paste your license key first';
input.focus();
return;
}
btn.disabled=true;
msg.className='trial-msg';
msg.textContent='Activating...';
try{
var resp=await fetch(A+'/license/activate',{
method:'POST',
headers:{'Content-Type':'application/json'},
body:JSON.stringify({license_key:key})
});
var data=await resp.json();
if(!resp.ok){
msg.className='trial-msg error';
msg.textContent=data.error||'Activation failed';
btn.disabled=false;
return;
}
msg.className='trial-msg success';
msg.textContent='Activated. Reloading...';
// Brief pause so the user sees the success message before the page reloads
setTimeout(function(){location.reload()},800);
}catch(e){
msg.className='trial-msg error';
msg.textContent='Network error: '+e.message;
btn.disabled=false;
}
}

// Allow Enter to submit the activation form
document.addEventListener('DOMContentLoaded',function(){
var input=document.getElementById('trial-key-input');
if(input){
input.addEventListener('keydown',function(e){
if(e.key==='Enter')activateLicense();
});
}
});
</script>
</body>
</html>`
