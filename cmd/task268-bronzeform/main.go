// task268-bronzeform 金文构形演变证据复核台
// 入口：--addr 监听地址，--db 数据库路径，--smoke-test 端到端自检。
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"task268-bronzeform/internal/httpapi"
	"task268-bronzeform/internal/service"
	"task268-bronzeform/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "bronzeform.db", "sqlite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	svc := service.New(
		store.NewBatchStore(db),
		store.NewGlyphStore(db),
		store.NewRelationStore(db),
		store.NewVersionStore(db),
	)

	if *smoke {
		if err := runSmokeTest(svc); err != nil {
			log.Fatalf("smoke test failed: %v", err)
		}
		os.Stdout.WriteString("smoke test passed\n")
		return
	}

	srv := &http.Server{
		Addr:    *addr,
		Handler: httpapi.New(svc).Handler(),
	}
	log.Printf("task268-bronzeform listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
