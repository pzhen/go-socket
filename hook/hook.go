package hook

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// 统计时间
func HookTime(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		defer func() {
			log.Printf("[Info] %s %s took %v", r.Method, r.URL.String(), time.Now().Sub(begin))
		}()
		fn(w, r)
	}
}

// 恐慌恢复
func HookRecover(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println("[Recover] recovered from runtime error:", err)
				debug.PrintStack()
			}
		}()
		fn(w, r)
	}
}