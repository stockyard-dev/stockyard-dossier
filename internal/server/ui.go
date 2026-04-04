package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Dossier</title>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--blue:#5b8dd9;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--serif);line-height:1.6}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-family:var(--mono);font-size:.9rem;letter-spacing:2px}.hdr h1 span{color:var(--rust)}
.main{padding:1.5rem;max-width:960px;margin:0 auto}
.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:.5rem;margin-bottom:1rem}
.st{background:var(--bg2);border:1px solid var(--bg3);padding:.6rem;text-align:center;font-family:var(--mono)}
.st-v{font-size:1.3rem;font-weight:700}.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.15rem}
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
.card-contact a{color:var(--blue);text-decoration:none}.card-contact a:hover{color:var(--rust)}
.card-meta{font-family:var(--mono);font-size:.55rem;color:var(--cm);margin-top:.35rem;display:flex;gap:.4rem;flex-wrap:wrap;align-items:center}
.card-notes{font-size:.7rem;color:var(--cm);margin-top:.35rem;font-style:italic;padding:.3rem .5rem;border-left:2px solid var(--bg3)}
.tag{font-family:var(--mono);font-size:.5rem;padding:.1rem .3rem;background:var(--bg3);color:var(--cd)}
.badge{font-family:var(--mono);font-size:.5rem;padding:.12rem .35rem;text-transform:uppercase;letter-spacing:1px;border:1px solid}
.badge.active{border-color:var(--green);color:var(--green)}.badge.lead{border-color:var(--gold);color:var(--gold)}.badge.inactive{border-color:var(--cm);color:var(--cm)}.badge.client{border-color:var(--blue);color:var(--blue)}
.card-actions{display:flex;gap:.3rem;flex-shrink:0}
.btn{font-family:var(--mono);font-size:.6rem;padding:.25rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .2s}
.btn:hover{border-color:var(--leather);color:var(--cream)}.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}
.btn-sm{font-size:.55rem;padding:.2rem .4rem}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:460px;max-width:92vw;max-height:90vh;overflow-y:auto}
.modal h2{font-family:var(--mono);font-size:.8rem;margin-bottom:1rem;color:var(--rust);letter-spacing:1px}
.fr{margin-bottom:.6rem}.fr label{display:block;font-family:var(--mono);font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.fr input:focus,.fr select:focus,.fr textarea:focus{outline:none;border-color:var(--leather)}
.row2{display:grid;grid-template-columns:1fr 1fr;gap:.5rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.85rem}
.count-label{font-family:var(--mono);font-size:.6rem;color:var(--cm);margin-bottom:.5rem}
@media(max-width:600px){.cards{grid-template-columns:1fr}.row2{grid-template-columns:1fr}.toolbar{flex-direction:column}.search{min-width:100%}}
</style></head><body>
<div class="hdr"><h1><span>&#9670;</span> DOSSIER</h1><button class="btn btn-p" onclick="openForm()">+ Add Contact</button></div>
<div class="main">
<div class="stats" id="stats"></div>
<div class="toolbar">
<input class="search" id="search" placeholder="Search name, email, company, tags..." oninput="render()">
<select class="filter-sel" id="status-filter" onchange="render()"><option value="">All Status</option><option value="active">Active</option><option value="lead">Lead</option><option value="client">Client</option><option value="inactive">Inactive</option></select>
</div>
<div class="count-label" id="count"></div>
<div class="cards" id="contacts"></div>
</div>
<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()"><div class="modal" id="mdl"></div></div>
<script>
var A='/api',contacts=[],editId=null;

async function load(){var r=await fetch(A+'/contacts').then(function(r){return r.json()});contacts=r.contacts||[];renderStats();render();}

function renderStats(){
var total=contacts.length;
var companies={};contacts.forEach(function(c){if(c.company)companies[c.company]=true});
var withEmail=contacts.filter(function(c){return c.email}).length;
document.getElementById('stats').innerHTML=[
{l:'Contacts',v:total},{l:'Companies',v:Object.keys(companies).length},{l:'With Email',v:withEmail}
].map(function(x){return '<div class="st"><div class="st-v">'+x.v+'</div><div class="st-l">'+x.l+'</div></div>'}).join('');
}

function render(){
var q=(document.getElementById('search').value||'').toLowerCase();
var sf=document.getElementById('status-filter').value;
var f=contacts;
if(sf)f=f.filter(function(c){return c.status===sf});
if(q)f=f.filter(function(c){return(c.name||'').toLowerCase().includes(q)||(c.email||'').toLowerCase().includes(q)||(c.company||'').toLowerCase().includes(q)||(c.tags||'').toLowerCase().includes(q)});
document.getElementById('count').textContent=f.length+' contact'+(f.length!==1?'s':'');
if(!f.length){document.getElementById('contacts').innerHTML='<div class="empty">No contacts found.</div>';return;}
var h='';f.forEach(function(c){
h+='<div class="card"><div class="card-top"><div>';
h+='<div class="card-name">'+esc(c.name)+'</div>';
if(c.company||c.title){h+='<div class="card-role">';if(c.title)h+=esc(c.title);if(c.title&&c.company)h+=' at ';if(c.company)h+=esc(c.company);h+='</div>';}
h+='</div><div class="card-actions">';
h+='<button class="btn btn-sm" onclick="openEdit(\''+c.id+'\')">Edit</button>';
h+='<button class="btn btn-sm" onclick="del(\''+c.id+'\')" style="color:var(--red)">&#10005;</button>';
h+='</div></div>';
h+='<div class="card-contact">';
if(c.email)h+='<a href="mailto:'+esc(c.email)+'">'+esc(c.email)+'</a>';
if(c.phone)h+='<span>'+esc(c.phone)+'</span>';
h+='</div>';
h+='<div class="card-meta">';
if(c.status)h+='<span class="badge '+c.status+'">'+c.status+'</span>';
if(c.tags){c.tags.split(',').forEach(function(t){t=t.trim();if(t)h+='<span class="tag">#'+esc(t)+'</span>';});}
h+='</div>';
if(c.notes)h+='<div class="card-notes">'+esc(c.notes)+'</div>';
h+='</div>';
});
document.getElementById('contacts').innerHTML=h;
}

async function del(id){if(!confirm('Remove this contact?'))return;await fetch(A+'/contacts/'+id,{method:'DELETE'});load();}

function formHTML(contact){
var i=contact||{name:'',email:'',phone:'',company:'',title:'',tags:'',notes:'',status:'active'};
var isEdit=!!contact;
var h='<h2>'+(isEdit?'EDIT CONTACT':'NEW CONTACT')+'</h2>';
h+='<div class="fr"><label>Name *</label><input id="f-name" value="'+esc(i.name)+'" placeholder="Full name"></div>';
h+='<div class="row2"><div class="fr"><label>Email</label><input id="f-email" type="email" value="'+esc(i.email)+'"></div>';
h+='<div class="fr"><label>Phone</label><input id="f-phone" value="'+esc(i.phone)+'"></div></div>';
h+='<div class="row2"><div class="fr"><label>Company</label><input id="f-company" value="'+esc(i.company)+'"></div>';
h+='<div class="fr"><label>Title / Role</label><input id="f-title" value="'+esc(i.title)+'"></div></div>';
h+='<div class="row2"><div class="fr"><label>Status</label><select id="f-status">';
['active','lead','client','inactive'].forEach(function(s){h+='<option value="'+s+'"'+(i.status===s?' selected':'')+'>'+s.charAt(0).toUpperCase()+s.slice(1)+'</option>';});
h+='</select></div><div class="fr"><label>Tags</label><input id="f-tags" value="'+esc(i.tags)+'" placeholder="comma separated"></div></div>';
h+='<div class="fr"><label>Notes</label><textarea id="f-notes" rows="3" placeholder="Notes about this contact...">'+esc(i.notes)+'</textarea></div>';
h+='<div class="acts"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-p" onclick="submit()">'+(isEdit?'Save':'Add Contact')+'</button></div>';
return h;
}

function openForm(){editId=null;document.getElementById('mdl').innerHTML=formHTML();document.getElementById('mbg').classList.add('open');document.getElementById('f-name').focus();}
function openEdit(id){var c=null;for(var j=0;j<contacts.length;j++){if(contacts[j].id===id){c=contacts[j];break;}}if(!c)return;editId=id;document.getElementById('mdl').innerHTML=formHTML(c);document.getElementById('mbg').classList.add('open');}
function closeModal(){document.getElementById('mbg').classList.remove('open');editId=null;}

async function submit(){
var name=document.getElementById('f-name').value.trim();
if(!name){alert('Name is required');return;}
var body={name:name,email:document.getElementById('f-email').value.trim(),phone:document.getElementById('f-phone').value.trim(),company:document.getElementById('f-company').value.trim(),title:document.getElementById('f-title').value.trim(),status:document.getElementById('f-status').value,tags:document.getElementById('f-tags').value.trim(),notes:document.getElementById('f-notes').value.trim()};
if(editId){await fetch(A+'/contacts/'+editId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}
else{await fetch(A+'/contacts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}
closeModal();load();
}

function esc(s){if(!s)return'';var d=document.createElement('div');d.textContent=s;return d.innerHTML;}
document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal();});
load();
</script></body></html>`
