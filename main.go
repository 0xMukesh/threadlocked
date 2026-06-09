package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	_ "embed"

	"github.com/syumai/workers"
)

type WebringEntry struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Gh   string `json:"gh"`
}

//go:embed webring.json
var webringRaw []byte

//go:embed index.html
var indexHTML []byte

var hostsToIgnore = []string{"ring.seggs.lol", "seggs.lol", "www.seggs.lol", "links.seggs.lol"}

// initial returns the uppercased first character of s (for the letter avatar).
func initial(s string) string {
	for _, r := range s {
		return strings.ToUpper(string(r))
	}
	return ""
}

// host strips the scheme and a leading "www." from a url for display.
func host(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return strings.TrimPrefix(u.Host, "www.")
}

var founderNames = map[string]bool{"datavorous": true, "nisarga": true}

func buildCards(entries []WebringEntry) string {
	var b strings.Builder
	for _, e := range entries {
		name := html.EscapeString(e.Name)
		b.WriteString(`<div class="card" data-name="`)
		b.WriteString(name)
		b.WriteString(`"><a class="card-link" href="`)
		b.WriteString(html.EscapeString(e.Url))
		b.WriteString(`" rel="noopener"><span class="avatar">`)
		b.WriteString(html.EscapeString(initial(e.Name)))
		if e.Gh != "" {
			b.WriteString(`<img src="https://avatars.githubusercontent.com/`)
			b.WriteString(html.EscapeString(e.Gh))
			b.WriteString(`?size=96" alt="" loading="lazy" onerror="this.setAttribute('data-failed','')" />`)
		}
		b.WriteString(`</span><span class="meta"><span class="name">`)
		b.WriteString(name)
		b.WriteString(`</span><span class="host">`)
		b.WriteString(html.EscapeString(host(e.Url)))
		b.WriteString(`</span></span><span class="arrow">&rarr;</span></a>`)
		if e.Gh != "" {
			gh := html.EscapeString(e.Gh)
			b.WriteString(`<a class="gh-link" href="https://github.com/`)
			b.WriteString(gh)
			b.WriteString(`" target="_blank" rel="noopener" aria-label="`)
			b.WriteString(name)
			b.WriteString(` on GitHub"><svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true" width="16" height="16"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg></a>`)
		}
		b.WriteString(`</div>`)
	}
	return b.String()
}

func renderIndex(founders, members []WebringEntry) []byte {
	out := strings.Replace(string(indexHTML), "<!--FOUNDERS-->", buildCards(founders), 1)
	out = strings.Replace(out, "<!--MEMBERS-->", buildCards(members), 1)
	out = strings.Replace(out, "<!--COUNT-->", fmt.Sprintf("%d members", len(founders)+len(members)), 1)
	return []byte(out)
}

func main() {
	var webring []WebringEntry
	if err := json.Unmarshal(webringRaw, &webring); err != nil {
		slog.Error("failed to unmarshal webring json file", "error", err)
		os.Exit(1)
	}

	var founders, members []WebringEntry
	for _, e := range webring {
		if founderNames[e.Name] {
			founders = append(founders, e)
		} else {
			members = append(members, e)
		}
	}

	page := renderIndex(founders, members)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqHost := r.Host

		if strings.HasSuffix(reqHost, ".seggs.lol") && !slices.Contains(hostsToIgnore, reqHost) {
			sub := strings.TrimSuffix(reqHost, ".seggs.lol")
			for _, entry := range webring {
				if entry.Name == sub {
					http.Redirect(w, r, entry.Url, http.StatusFound)
					return
				}
			}

			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})

	http.HandleFunc("/webring", func(w http.ResponseWriter, r *http.Request) {
		buildJsonResponse(w, http.StatusOK, webring)
	})

	http.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		if from == "" {
			buildJsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "missing `from` query parameter",
			})
			return
		}

		dir := r.URL.Query().Get("dir")
		if dir == "" {
			buildJsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "missing `dir` query parameter",
			})
			return
		}

		if dir != "next" && dir != "prev" {
			buildJsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "invalid `dir` query parameter. it can be either `next` or `prev` only",
			})
			return
		}

		index := -1
		for i, v := range webring {
			if v.Name == from {
				index = i
				break
			}
		}

		if index == -1 {
			buildJsonResponse(w, http.StatusBadRequest, map[string]string{
				"error": "invalid `from` query parameter. can't find any webring entry's name as `from`",
			})
			return
		}

		url := ""

		if dir == "prev" {
			if index == 0 {
				url = webring[len(webring)-1].Url
			} else {
				url = webring[index-1].Url
			}
		} else {
			if index == len(webring)-1 {
				url = webring[0].Url
			} else {
				url = webring[index+1].Url
			}
		}

		setCorsHeaders(w)
		http.Redirect(w, r, url, http.StatusFound)
	})

	http.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		index := rand.Intn(len(webring))
		http.Redirect(w, r, webring[index].Url, http.StatusFound)
	})

	workers.Serve(nil)
	fmt.Println("server is up and running at :8080")
}

func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func buildJsonResponse(w http.ResponseWriter, statusCode int, v any) {
	setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}
