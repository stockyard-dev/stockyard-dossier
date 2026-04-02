package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Document struct{
	ID string `json:"id"`
	Title string `json:"title"`
	Body string `json:"body"`
	Category string `json:"category"`
	Tags string `json:"tags"`
	Author string `json:"author"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"dossier.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS documents(id TEXT PRIMARY KEY,title TEXT NOT NULL,body TEXT DEFAULT '',category TEXT DEFAULT '',tags TEXT DEFAULT '',author TEXT DEFAULT '',status TEXT DEFAULT 'draft',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Document)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO documents(id,title,body,category,tags,author,status,created_at)VALUES(?,?,?,?,?,?,?,?)`,e.ID,e.Title,e.Body,e.Category,e.Tags,e.Author,e.Status,e.CreatedAt);return err}
func(d *DB)Get(id string)*Document{var e Document;if d.db.QueryRow(`SELECT id,title,body,category,tags,author,status,created_at FROM documents WHERE id=?`,id).Scan(&e.ID,&e.Title,&e.Body,&e.Category,&e.Tags,&e.Author,&e.Status,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Document{rows,_:=d.db.Query(`SELECT id,title,body,category,tags,author,status,created_at FROM documents ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Document;for rows.Next(){var e Document;rows.Scan(&e.ID,&e.Title,&e.Body,&e.Category,&e.Tags,&e.Author,&e.Status,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM documents WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&n);return n}
