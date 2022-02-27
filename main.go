package main

import (
	"context"
	"fmt"
	"github.com/aragorn-yang/go-camp-04/rate_limiter"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	win, err := rate_limiter.StartWindow(time.Minute, 60, 10)
	if err != nil {
		log.Fatalln(err)
		return
	}

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	appHandler := http.NewServeMux()
	appHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// todo(bug): 每次刷新浏览器页面，会在服务器端触发两次 GET 请求
		// log.Printf("%s\n", r.Method)

		if win.Open() {
			w.Write([]byte("Reached rate limit\n"))
			w.Write([]byte("Current count is "))
			w.Write([]byte(strconv.Itoa(win.Count())))
			return
		}

		win.Increment()
		w.Write([]byte("Happy go camping\n"))
		w.Write([]byte("Current count is "))
		w.Write([]byte(strconv.Itoa(win.Count())))
	})

	app := http.Server{
		Handler: appHandler,
		Addr:    ":8080",
	}

	g.Go(func() error {
		return app.ListenAndServe()
	})

	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-quit:
			fmt.Printf("get signal %s, application is gonna shut down\n", sig)
			app.Shutdown(ctx)
			return fmt.Errorf("received os signal: %v\n", sig)
		}
	})

	err = g.Wait()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("context was canceled")
		} else {
			fmt.Printf("received error: %v\n", err)
		}
	} else {
		fmt.Println("finished clean")
	}
}
