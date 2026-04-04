package server
import "net/http"
func(s *Server)dashboard(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","text/html");w.Write([]byte(dashHTML))}
const dashHTML=`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Dossier</title>
<style>:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--mono:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-size:.9rem;letter-spacing:2px}
.main{padding:1.5rem;max-width:900px;margin:0 auto}
.search{width:100%;padding:.5rem .8rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.78rem;margin-bottom:1rem}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:.6rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1rem}
.card-name{font-size:.9rem;color:var(--cream);margin-bottom:.1rem}
.card-company{font-size:.7rem;color:var(--gold)}
.card-title{font-size:.65rem;color:var(--cm)}
.card-contact{font-size:.65rem;color:var(--cd);margin-top:.4rem}.card-contact a{color:var(--rust)}
.card-tags{margin-top:.3rem}
.tag{font-size:.5rem;padding:.1rem .3rem;background:var(--bg3);color:var(--cm);margin-right:.2rem}
.card-notes{font-size:.65rem;color:var(--cm);margin-top:.3rem;font-style:italic}
.btn{font-size:.6rem;padding:.25rem .6rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd)}.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:var(--bg)}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;align-items:center;justify-content:center}.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:400px;max-width:90vw}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust)}
.fr{margin-bottom:.5rem}.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.15rem}
.fr input,.fr textarea{width:100%;padding:.35rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:.8rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.75rem}
</style></head><body>
<div class="hdr"><h1>DOSSIER</h1><button class="btn btn-p" onclick="openForm()">+ Add Contact</button></div>
<div class="main">
<input class="search" id="search" placeholder="Search contacts..." oninput="render()">
<div class="grid" id="grid"></div>
</div>
<div class="modal-bg" id="mbg" onclick="if(event.target===this)cm()"><div class="modal" id="mdl"></div></div>
<script>
const A='/api';let contacts=[];
async function load(){const r=await fetch(A+'/contacts').then(r=>r.json());contacts=r.contacts||[];render();}
function render(){const q=(document.getElementById('search').value||'').toLowerCase();
let filtered=contacts.filter(c=>!q||(c.name+c.company+c.email+c.tags).toLowerCase().includes(q));
if(!filtered.length){document.getElementById('grid').innerHTML='<div class="empty">No contacts'+(q?' matching':'')+'.</div>';return;}
let h='';filtered.forEach(c=>{
h+='<div class="card"><div style="display:flex;justify-content:space-between"><div class="card-name">'+esc(c.name)+'</div><button class="btn" onclick="del(\''+c.id+'\')" style="font-size:.5rem;color:var(--cm)">✕</button></div>';
if(c.company)h+='<div class="card-company">'+esc(c.company)+'</div>';
if(c.title)h+='<div class="card-title">'+esc(c.title)+'</div>';
h+='<div class="card-contact">';if(c.email)h+='<a href="mailto:'+c.email+'">'+esc(c.email)+'</a><br>';if(c.phone)h+=esc(c.phone);h+='</div>';
if(c.tags){const tags=c.tags.split(',');h+='<div class="card-tags">';tags.forEach(t=>{if(t.trim())h+='<span class="tag">'+esc(t.trim())+'</span>';});h+='</div>';}
if(c.notes)h+='<div class="card-notes">'+esc(c.notes.substring(0,80))+'</div>';
h+='</div>';});
document.getElementById('grid').innerHTML=h;}
async function del(id){if(confirm('Delete?')){await fetch(A+'/contacts/'+id,{method:'DELETE'});load();}}
function openForm(){document.getElementById('mdl').innerHTML='<h2>Add Contact</h2><div class="fr"><label>Name</label><input id="f-n"></div><div class="fr"><label>Email</label><input id="f-e" type="email"></div><div class="fr"><label>Phone</label><input id="f-p"></div><div class="fr"><label>Company</label><input id="f-co"></div><div class="fr"><label>Title</label><input id="f-t"></div><div class="fr"><label>Tags (comma separated)</label><input id="f-tg" placeholder="lead, partner, vendor"></div><div class="fr"><label>Notes</label><textarea id="f-nt" rows="2"></textarea></div><div class="acts"><button class="btn" onclick="cm()">Cancel</button><button class="btn btn-p" onclick="sub()">Add</button></div>';document.getElementById('mbg').classList.add('open');}
async function sub(){await fetch(A+'/contacts',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:document.getElementById('f-n').value,email:document.getElementById('f-e').value,phone:document.getElementById('f-p').value,company:document.getElementById('f-co').value,title:document.getElementById('f-t').value,tags:document.getElementById('f-tg').value,notes:document.getElementById('f-nt').value})});cm();load();}
function cm(){document.getElementById('mbg').classList.remove('open');}
function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
load();
</script></body></html>`
