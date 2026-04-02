package main
import ("fmt";"log";"net/http";"os";"github.com/stockyard-dev/stockyard-dossier/internal/server";"github.com/stockyard-dev/stockyard-dossier/internal/store")
func main(){port:=os.Getenv("PORT");if port==""{port="9080"};dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./dossier-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("dossier: %v",err)};defer db.Close();srv:=server.New(db)
fmt.Printf("\n  Dossier — contact and CRM manager\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n\n",port,port)
log.Printf("dossier: listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,srv))}
