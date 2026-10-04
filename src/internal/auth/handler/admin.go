package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/elecbug/linuxus/src/internal/common/user"
)

func (a *App) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	id, ok := a.getSessionID(r)
	if !ok {
		http.Error(w, "authentication required", 401)
		return false
	}
	if a.adminID == "" || id != a.adminID {
		http.Error(w, "administrator access required", 403)
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	return true
}

func (a *App) adminToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return a.sign("admin-csrf|" + cookie.Value)
}

func (a *App) managerOperation(ctx context.Context, path, method string, payload any, result any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.managerBaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Manager-Session-Secret", a.managerSessionSecret)
	resp, err := a.managerClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Manager operation failed (HTTP %d)", resp.StatusCode)
	}
	if result != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(result); err != nil {
			return err
		}
		if warning := resp.Header.Get("X-Linuxus-Disk-Error"); warning != "" {
			return fmt.Errorf("%s", warning)
		}
	}
	return nil
}

func (a *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !a.requireAdmin(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	if err := adminTemplate.Execute(w, map[string]string{"Token": a.adminToken(r)}); err != nil {
		return
	}
}

func (a *App) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		accounts, err := user.LoadUsers(a.authListFile)
		if err != nil {
			http.Error(w, "cannot read accounts", 500)
			return
		}
		states := map[string]map[string]any{}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		stateErr := a.managerOperation(ctx, "/admin/status", http.MethodGet, nil, &states)
		ids := make([]string, 0, len(accounts))
		for id := range accounts {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		rows := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			account, err := user.ParseAccount(accounts[id])
			if err != nil {
				http.Error(w, "invalid account record", 500)
				return
			}
			row := states[id]
			if row == nil {
				row = map[string]any{}
			}
			row["id"] = id
			row["locked"] = account.Locked || account.Maintenance
			row["maintenance"] = account.Maintenance
			row["class"] = account.Class
			row["template"] = account.Template
			rows = append(rows, row)
		}
		result := map[string]any{"users": rows, "templates": a.templates, "classes": a.classes}
		if stateErr != nil {
			result["warning"] = stateErr.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if token := r.Header.Get("X-CSRF-Token"); token == "" || !hmac.Equal([]byte(token), []byte(a.adminToken(r))) {
		http.Error(w, "invalid CSRF token", 403)
		return
	}
	var request struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
		Value  string `json:"value"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	action := request.Action
	switch action {
	case "lock":
		if request.UserID == a.adminID {
			http.Error(w, "cannot lock the current administrator", 400)
			return
		}
	case "unlock", "password", "disconnect", "restart":
	case "template":
		if _, ok := a.templates[request.Value]; !ok && request.Value != "default" {
			http.Error(w, "unknown template", 400)
			return
		}
	case "class":
		if _, ok := a.classes[request.Value]; !ok {
			http.Error(w, "unknown class", 400)
			return
		}
	default:
		http.Error(w, "unsupported operation", 400)
		return
	}
	if action == "restart" {
		action = "disconnect"
	}
	if err := user.UpdateAccount(a.authListFile, request.UserID, action, request.Value); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if action != "unlock" {
		ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
		defer cancel()
		operation := "stop"
		if request.Action == "restart" {
			operation = "restart"
		}
		if err := a.managerOperation(ctx, "/admin/user", http.MethodPost, map[string]string{"user_id": request.UserID, "action": operation}, nil); err != nil {
			http.Error(w, "Account updated; "+err.Error(), 503)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"ok":true}`)
}

var adminTemplate = template.Must(template.New("admin").Parse(`<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Linuxus Administration</title>
<style>body{font:16px system-ui;margin:2rem;background:#f5f7fb;color:#18243b}main{max-width:1200px;margin:auto}table{width:100%;border-collapse:collapse;background:white}th,td{padding:.8rem;text-align:left;border-bottom:1px solid #ddd}button,select{padding:.45rem;margin:.15rem}#message{white-space:pre-wrap;color:#8b241a}small{color:#506078}.scroll{overflow:auto}</style>
<main><h1>Linuxus Administration</h1><p>Locking accounts or disconnecting sessions preserves exercise data.</p><button id="refresh">Refresh</button><p id="message" role="status"></p><div class="scroll"><table><thead><tr><th>User</th><th>Account and sessions</th><th>Disk usage</th><th>Classroom environment</th><th>Actions</th></tr></thead><tbody id="users"></tbody></table></div></main>
<script>
const token={{.Token}}, message=document.getElementById('message'), tbody=document.getElementById('users');
const size=n=>n===undefined?'—':(n/1024/1024/1024).toFixed(2)+' GiB';
async function action(id,op,value='') {
 if(!confirm('Apply '+op+' to '+id+'?'))return;
 try {const r=await fetch('/admin/api/users',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':token},body:JSON.stringify({user_id:id,action:op,value})});if(!r.ok)throw Error(await r.text());await refresh();}catch(e){message.textContent=e.message;}
}
function button(cell,label,id,op){const b=document.createElement('button');b.textContent=label;b.onclick=()=>action(id,op);cell.append(b);}
function choice(cell,items,current,id,op){const s=document.createElement('select');const placeholder=document.createElement('option');placeholder.value='';placeholder.textContent=op==='class'?'Select class':'Select template';s.append(placeholder);for(const name of items){const o=document.createElement('option');o.value=name;o.textContent=name;o.selected=name===current;s.append(o)}s.onchange=()=>{if(s.value)action(id,op,s.value)};cell.append(s);}
async function refresh(){try{const r=await fetch('/admin/api/users',{cache:'no-store'});if(!r.ok)throw Error(await r.text());const data=await r.json();message.textContent=data.warning||'';tbody.replaceChildren();for(const u of data.users){const tr=document.createElement('tr');const cells=Array.from({length:5},()=>{const c=document.createElement('td');tr.append(c);return c});cells[0].textContent=u.id;cells[1].textContent=(u.locked?'Locked':'Enabled')+' / '+(u.state||'Stopped')+' / Sessions: '+(u.sessions||0);cells[2].textContent=u.disk_error||(u.mounted?size(u.used_bytes)+' / '+size(u.total_bytes):'Not mounted');choice(cells[3],['default',...Object.keys(data.templates||{})],u.template,u.id,'template');choice(cells[3],Object.keys(data.classes||{}),u.class,u.id,'class');button(cells[4],u.locked?'Unlock':'Lock',u.id,u.locked?'unlock':'lock');button(cells[4],'Disconnect',u.id,'disconnect');button(cells[4],'Restart environment',u.id,'restart');const reset=document.createElement('button');reset.textContent='Reset password';reset.onclick=()=>{document.getElementById('password-user').value=u.id;document.getElementById('password-dialog').showModal()};cells[4].append(reset);tbody.append(tr)}}catch(e){message.textContent=e.message}}
document.getElementById('refresh').onclick=refresh;refresh();setInterval(refresh,15000);
</script><dialog id="password-dialog"><form method="dialog"><h2>Reset password</h2><input id="password-user" readonly aria-label="User"><input id="new-password" type="password" autocomplete="new-password" required aria-label="New password"><button id="save-password">Save</button><button formnovalidate>Cancel</button></form></dialog><script>document.getElementById('save-password').onclick=e=>{e.preventDefault();const input=document.getElementById('new-password');if(!input.value)return;action(document.getElementById('password-user').value,'password',input.value);input.value='';document.getElementById('password-dialog').close()}</script></html>`))
