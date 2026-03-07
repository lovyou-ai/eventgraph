// Command eg is a CLI for interacting with an EventGraph store.
//
// Usage:
//
//	eg bootstrap              — Initialize a new event graph
//	eg get <event-id>         — Get an event by ID
//	eg recent [limit]         — Show recent events
//	eg count                  — Show event count
//	eg verify                 — Verify hash chain integrity
//	eg head                   — Show the chain head
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/lovyou-ai/eventgraph/go/pkg/actor"
	"github.com/lovyou-ai/eventgraph/go/pkg/event"
	"github.com/lovyou-ai/eventgraph/go/pkg/graph"
	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/types"
)

type noopSigner struct{}

func (noopSigner) Sign(data []byte) (types.Signature, error) {
	sig := make([]byte, 64)
	copy(sig, data[:min(64, len(data))])
	return types.MustSignature(sig), nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	s := store.NewInMemoryStore()
	as := actor.NewInMemoryActorStore()
	g := graph.New(s, as)
	if err := g.Start(); err != nil {
		fatal("graph start failed: %v", err)
	}
	defer g.Close()

	switch cmd {
	case "bootstrap":
		cmdBootstrap(g)
	case "get":
		if len(os.Args) < 3 {
			fatal("usage: eg get <event-id>")
		}
		cmdGet(g, os.Args[2])
	case "recent":
		limit := 10
		if len(os.Args) >= 3 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil {
				fatal("invalid limit: %s", os.Args[2])
			}
			limit = n
		}
		cmdRecent(g, limit)
	case "count":
		cmdCount(g)
	case "verify":
		cmdVerify(g)
	case "head":
		cmdHead(g)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func cmdBootstrap(g *graph.Graph) {
	actorID := types.MustActorID("actor_system0000000000000000000001")
	ev, err := g.Bootstrap(actorID, noopSigner{})
	if err != nil {
		fatal("bootstrap failed: %v", err)
	}
	printEvent(ev)
}

func cmdGet(g *graph.Graph, idStr string) {
	id, err := types.NewEventID(idStr)
	if err != nil {
		fatal("invalid event ID: %v", err)
	}
	ev, err := g.Store().Get(id)
	if err != nil {
		fatal("get failed: %v", err)
	}
	printEvent(ev)
}

func cmdRecent(g *graph.Graph, limit int) {
	q, err := g.Query()
	if err != nil {
		fatal("query: %v", err)
	}
	page, err := q.Recent(limit)
	if err != nil {
		fatal("recent failed: %v", err)
	}
	for _, ev := range page.Items() {
		printEventSummary(ev)
	}
	if len(page.Items()) == 0 {
		fmt.Println("(no events)")
	}
}

func cmdCount(g *graph.Graph) {
	q, err := g.Query()
	if err != nil {
		fatal("query: %v", err)
	}
	count, err := q.EventCount()
	if err != nil {
		fatal("count failed: %v", err)
	}
	fmt.Printf("%d events\n", count)
}

func cmdVerify(g *graph.Graph) {
	result, err := g.Store().VerifyChain()
	if err != nil {
		fatal("verify failed: %v", err)
	}
	fmt.Printf("Chain verified: %d events, valid=%v\n", result.Length, result.Valid)
}

func cmdHead(g *graph.Graph) {
	head, err := g.Store().Head()
	if err != nil {
		fatal("head failed: %v", err)
	}
	if !head.IsSome() {
		fmt.Println("(empty chain)")
		return
	}
	printEvent(head.Unwrap())
}

func printEvent(ev event.Event) {
	out := map[string]any{
		"id":              ev.ID().Value(),
		"type":            ev.Type().Value(),
		"source":          ev.Source().Value(),
		"timestamp":       ev.Timestamp().String(),
		"hash":            ev.Hash().Value(),
		"prev_hash":       ev.PrevHash().Value(),
		"conversation_id": ev.ConversationID().Value(),
		"version":         ev.Version(),
		"is_bootstrap":    ev.IsBootstrap(),
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func printEventSummary(ev event.Event) {
	fmt.Printf("  %s  %s  %s\n", ev.ID().Value(), ev.Type().Value(), ev.Timestamp().String())
}

func usage() {
	fmt.Println(`eg — EventGraph CLI

Usage:
  eg bootstrap              Initialize a new event graph
  eg get <event-id>         Get an event by ID
  eg recent [limit]         Show recent events (default: 10)
  eg count                  Show event count
  eg verify                 Verify hash chain integrity
  eg head                   Show the chain head
  eg help                   Show this help`)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
