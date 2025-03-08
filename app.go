package main

import (
	"context"
	"fmt"
	"log"

	libp2p "github.com/libp2p/go-libp2p"
)

//! TEST FUNCTION TO BE REMOVED
func Server() {
	h, err := libp2p.New()
	if err != nil {
		panic(err)
	}
	defer h.Close()
	log.Printf("Hello World, my hosts ID is %s\n", h.ID())

}

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	Server()
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
