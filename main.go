package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/feeds"
	"suah.dev/protect"
)

var search string
var prefix string

// GQLQuery is what github wants in the POST request.
type GQLQuery struct {
	Query string `json:"query"`
}

// GHResp represents a response from GitHub's GraphQL API.
type GHResp struct {
	Data Data `json:"data"`
}

// PageInfo represents the page information.
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// Repository information has repo specific bits.
type Repository struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	StargazerCount int    `json:"stargazerCount"`
}

// Node is an entry from our "search".
type Node struct {
	Repository Repository `json:"repository"`
	CreatedAt  time.Time  `json:"createdAt"`
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	Author     Author     `json:"author"`
	BodyHTML   string     `json:"bodyHTML"`
}

// Edges are ... too edgy to tell..
type Edges struct {
	Node Node `json:"node,omitempty"`
}

// Search bundles our edges together
type Search struct {
	IssueCount int      `json:"issueCount"`
	PageInfo   PageInfo `json:"pageInfo"`
	Edges      []Edges  `json:"edges"`
}

// Author is an individual author
type Author struct {
	Login string `json:"login"`
	URL   string `json:"url"`
}

// Data is the data returned from a search
type Data struct {
	Search Search `json:"search"`
}

const endPoint = "https://api.github.com/graphql"
const ghQuery = `
{
  search(first: 100, type: ISSUE, query: "state:open %s") {
    issueCount
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        ... on Issue {
          repository {
            name
            url
            stargazerCount
          }
          createdAt
          title
          url
	  bodyHTML
	  author {
            login
            url
	  }
        }
      }
    }
  }
}
`

func getData(q GQLQuery) (re *GHResp, err error) {
	var req *http.Request
	client := &http.Client{}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(q); err != nil {
		return nil, err
	}

	req, err = http.NewRequest("POST", endPoint, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", os.Getenv("GH_AUTH_TOKEN")))

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(data, &re); err != nil {
		return nil, err
	}

	return re, nil
}

func makeRSS(q *GHResp) {
	feed := &feeds.Feed{
		Title:       fmt.Sprintf("%s GitHub Issues", search),
		Link:        &feeds.Link{Href: "https://github.com/qbit/gqrss"},
		Description: fmt.Sprintf("Open GitHub issues relating to %s", search),
		Author:      &feeds.Author{Name: "Aaron Bieber", Email: "aaron@bolddaemon.com"},
		Copyright:   "This work is copyright © Aaron Bieber",
	}

	for _, e := range q.Data.Search.Edges {
		if e.Node.Title == "" {
			continue
		}

		f := &feeds.Item{
			Title:       fmt.Sprintf("%s: %s", e.Node.Repository.Name, e.Node.Title),
			Link:        &feeds.Link{Href: e.Node.URL},
			Created:     e.Node.CreatedAt,
			Description: e.Node.BodyHTML,
			Author: &feeds.Author{
				Name: e.Node.Author.Login,
			},
		}

		feed.Items = append(feed.Items, f)
	}

	atomFile, err := os.Create(fmt.Sprintf("%satom.xml", prefix))
	if err != nil {
		log.Fatal(err)
	}

	rssFile, err := os.Create(fmt.Sprintf("%srss.xml", prefix))
	if err != nil {
		log.Fatal(err)
	}

	feed.WriteAtom(atomFile)
	feed.WriteRss(rssFile)
}

func main() {
	flag.StringVar(&search, "search", "OpenBSD", "thing to search GitHub for")
	flag.StringVar(&prefix, "prefix", "", "prefix to prepend to file names")
	flag.Parse()
	protect.Unveil("./", "rwc")
	protect.Unveil("/etc/ssl/cert.pem", "r")
	protect.Pledge("stdio unveil rpath wpath cpath flock dns inet")

	protect.UnveilBlock()

	var q GQLQuery
	q.Query = fmt.Sprintf(ghQuery, search)

	resp, err := getData(q)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	makeRSS(resp)
}
